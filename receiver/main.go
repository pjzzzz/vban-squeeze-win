// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
//
// Receives a VBAN audio stream, normalises it to 16-bit signed stereo PCM at
// 44100 Hz (what Lyrion's "pcm" stream format expects) and writes it to stdout.
// Lyrion's socketwrapper.exe forwards stdout to the Squeezebox.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const version = "1.0.0"

const usageText = `vban_url_receptor ` + version + `

Receive a VBAN audio stream and write 16-bit stereo PCM to stdout.

Usage:
  vban_url_receptor.exe [OPTIONS] vban://[host[:port]][/streamname]

URL defaults: host 127.0.0.1, port 6980, streamname Stream1.
The host part is informational only; the receiver listens on 0.0.0.0 and
selects the stream by name, like vban_receptor from quiniouben/vban.

Options:
  -p, --port=PORT        UDP port to listen on (default from URL, else 6980)
  -s, --streamname=NAME  VBAN stream name (default from URL, else Stream1)
  -q, --quality=0..4     receive quality hint (default 1)
  -c, --channels=LIST    channel list, e.g. 1,2 (default: automatic)
      --allow=IP[,IP]    only accept packets from these source addresses
      --interface=IP     local IPv4 address to join a multicast group on
  -d, --device=FILE      write PCM to FILE instead of stdout (debug, -o is an alias)
  -t, --seconds=N        stop after N seconds (testing)
      --stats            print a summary on stderr when stopping
      --title-file=FILE  write VBAN TEXT titles to FILE (default: temp/vban-<stream>.title)
  -l, --loglevel=0..4    log level: 0 fatal, 1 error, 2 info, 3 debug (default 2)
  -v, --version          print version
  -h, --help             print this help

Examples:
  vban_url_receptor.exe "vban://192.168.178.79/Stream1"
  vban_url_receptor.exe "vban://192.168.178.79:6980/Stream1" -d test.pcm -t 10 --stats
`

var (
	errHelp    = errors.New("help requested")
	errVersion = errors.New("version requested")
)

type options struct {
	url        string
	port       int
	stream     string
	quality    int
	channels   string
	channelMap []int
	allow      []string
	allowIPs   []net.IP
	iface      net.IP
	device     string
	seconds    float64
	stats      bool
	logLevel   int
	titleFile  string
}

func parseArgs(args []string) (options, error) {
	o := options{quality: 1, logLevel: levelInfo}
	var positional []string

	value := func(i *int, name, inline string, hasInline bool) (string, error) {
		if hasInline {
			return inline, nil
		}
		if *i+1 >= len(args) {
			return "", fmt.Errorf("option %s requires a value", name)
		}
		*i = *i + 1
		return args[*i], nil
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}

		name, inline, hasInline := arg, "", false
		if eq := strings.IndexByte(arg, '='); eq >= 0 {
			name, inline, hasInline = arg[:eq], arg[eq+1:], true
		}

		switch name {
		case "-h", "--help":
			return o, errHelp
		case "-v", "--version":
			return o, errVersion
		case "-p", "--port":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return o, fmt.Errorf("invalid port %q", v)
			}
			o.port = n
		case "-s", "--streamname":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			o.stream = v
		case "-q", "--quality":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > 4 {
				return o, fmt.Errorf("invalid quality %q (expected 0-4)", v)
			}
			o.quality = n
		case "-c", "--channels":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			o.channels = v
		case "--allow":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			o.allow = append(o.allow, strings.Split(v, ",")...)
		case "--interface":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			ip := net.ParseIP(strings.TrimSpace(v))
			if ip == nil || ip.To4() == nil {
				return o, fmt.Errorf("invalid IPv4 interface address %q", v)
			}
			o.iface = ip.To4()
		case "-d", "--device", "-o", "--output":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			o.device = v
		case "-t", "--seconds":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f <= 0 {
				return o, fmt.Errorf("invalid number of seconds %q", v)
			}
			o.seconds = f
		case "--stats":
			o.stats = true
		case "--title-file":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			o.titleFile = v
		case "-l", "--loglevel":
			v, err := value(&i, name, inline, hasInline)
			if err != nil {
				return o, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > 4 {
				return o, fmt.Errorf("invalid log level %q (expected 0-4)", v)
			}
			o.logLevel = n
		default:
			return o, fmt.Errorf("unknown option %q", name)
		}
	}

	if len(positional) > 1 {
		return o, fmt.Errorf("too many arguments: %s", strings.Join(positional[1:], " "))
	}
	if len(positional) == 1 {
		o.url = positional[0]
	}
	if o.url == "" {
		return o, errors.New("missing vban:// URL")
	}
	if o.port != 0 && (o.port < 1 || o.port > 65535) {
		return o, fmt.Errorf("port %d out of range", o.port)
	}
	if len(o.stream) > vbanStreamNameSize {
		return o, fmt.Errorf("stream name %q is longer than %d bytes", o.stream, vbanStreamNameSize)
	}

	channelMap, err := parseChannelMap(o.channels)
	if err != nil {
		return o, err
	}
	o.channelMap = channelMap

	for _, entry := range o.allow {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			return o, fmt.Errorf("invalid --allow address %q", entry)
		}
		o.allowIPs = append(o.allowIPs, ip)
	}

	return o, nil
}

func main() {
	opts, err := parseArgs(os.Args[1:])

	switch {
	case errors.Is(err, errHelp):
		fmt.Print(usageText)
		return
	case errors.Is(err, errVersion):
		fmt.Printf("vban_url_receptor %s\n", version)
		return
	case err != nil:
		fmt.Fprintf(os.Stderr, "vban_url_receptor: %v\n\n%s", err, usageText)
		os.Exit(2)
	}

	log := newLogger(opts.logLevel)

	stream, err := parseURL(opts.url)
	if err != nil {
		log.fatal("%v", err)
		os.Exit(2)
	}
	if opts.port != 0 {
		stream.Port = opts.port
	}
	if opts.stream != "" {
		stream.Stream = opts.stream
	}

	var out io.Writer = os.Stdout
	if opts.device != "" {
		f, err := os.Create(opts.device)
		if err != nil {
			log.fatal("cannot create %s: %v", opts.device, err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := receive(ctx, stream, opts, out, log)
	if err != nil {
		log.error("%v", err)
		os.Exit(1)
	}

	if opts.stats {
		printStats(os.Stderr, st)
	}
}

type stats struct {
	start        time.Time
	duration     time.Duration
	datagrams    int
	packets      int
	rejected     int
	otherStream  int
	blocked      int
	gaps         int
	bytesIn      int64
	bytesOut     int64
	silence      int64
	texts        int
	frames       int64
	peak         int
	sampleRate   int
	bitFmt       int
	bitLabel     string
	channels     int
	inRate       float64
	consumerGone bool
}

func receive(ctx context.Context, stream streamURL, opts options, out io.Writer, log *logger) (stats, error) {
	st := stats{start: time.Now()}

	conn, err := listenWithRetry(ctx, stream, opts, log)
	if err != nil {
		return st, err
	}
	defer conn.Close()

	log.info("listening on UDP %d for stream %q (source hint %s, quality %d)",
		stream.Port, stream.Stream, stream.Host, opts.quality)

	var deadline time.Time
	if opts.seconds > 0 {
		deadline = time.Now().Add(time.Duration(opts.seconds * float64(time.Second)))
	}

	buf := make([]byte, vbanMaxPacketSize)

	// A VBAN source can go silent (stream switched off). Keep the pipeline fed
	// with silence so Lyrion and the player do not mark the stream as failed;
	// real audio resumes as soon as packets arrive again.
	const silenceInterval = 100 * time.Millisecond
	silence := make([]byte, 44100*4*int(silenceInterval/time.Millisecond)/1000)
	lastWrite := time.Now()

	titlePath := opts.titleFile
	if titlePath == "" {
		name := "vban-title-" + safeFileName(stream.Stream) + ".txt"
		if st, err := os.Stat(`C:\ProgramData\Lyrion`); err == nil && st.IsDir() {
			titlePath = filepath.Join(`C:\ProgramData\Lyrion`, name)
		} else {
			titlePath = filepath.Join(os.TempDir(), name)
		}
	}
	lastTitle := ""

	res := newResampler(1.0)
	measureStart := time.Now()
	lastPacket := time.Now()
	var measureFrames int64
	firstPacket := true

	aw := newAsyncWriter(out)
	defer aw.close()

	var lastFrame uint32
	haveFrame := false
	announced := false
	warnedRate := false
	warnedChannels := false

	for {
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			break
		}

		select {
		case <-ctx.Done():
			st.duration = time.Since(st.start)
			return st, nil
		default:
		}

		readDeadline := time.Now().Add(silenceInterval)
		if !deadline.IsZero() && deadline.Before(readDeadline) {
			readDeadline = deadline
		}
		if err := conn.SetReadDeadline(readDeadline); err != nil {
			return st, err
		}

		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				// Only reset the rate measurement after a real gap; a short
				// timeout can also be caused by output backpressure.
				if time.Since(lastPacket) > time.Second {
					measureStart = time.Now()
					measureFrames = 0
				}

				if time.Since(lastWrite) >= silenceInterval {
					written := len(silence)
					werr := aw.write(silence)
					if werr != nil {
						if opts.device == "" && isClosedPipe(werr) {
							log.debug("output closed by consumer: %v", werr)
							st.consumerGone = true
							st.duration = time.Since(st.start)
							return st, nil
						}
						return st, fmt.Errorf("writing PCM output: %w", werr)
					}
					st.silence += int64(written)
					st.bytesOut += int64(written)
					lastWrite = time.Now()
				}
				continue
			}
			if ctx.Err() != nil {
				st.duration = time.Since(st.start)
				return st, nil
			}
			return st, err
		}

		st.datagrams++
		lastPacket = time.Now()

		if len(opts.allowIPs) > 0 && !sourceAllowed(addr.IP, opts.allowIPs) {
			st.blocked++
			continue
		}

		if name, text, ok := parseTextPacket(buf[:n]); ok {
			if name == stream.Stream && text != "" && text != lastTitle {
				if err := writeTitle(titlePath, text); err != nil {
					log.debug("cannot write title file %s: %v", titlePath, err)
				} else {
					lastTitle = text
					st.texts++
					log.info("title: %s", text)
				}
			}
			continue
		}

		pkt, err := parsePacket(buf[:n])
		if err != nil {
			st.rejected++
			if st.rejected <= 5 {
				log.debug("ignored packet from %s: %v", addr, err)
			}
			continue
		}

		if pkt.StreamName != stream.Stream {
			st.otherStream++
			continue
		}

		if !announced {
			announced = true
			st.sampleRate = pkt.SampleRate
			st.bitFmt = pkt.BitFmt
			st.bitLabel = pkt.bitDepthLabel()
			st.channels = pkt.Channels
			log.info("receiving stream %q: %d Hz / %s / %d ch from %s",
				pkt.StreamName, pkt.SampleRate, pkt.bitDepthLabel(), pkt.Channels, addr)
		}

		if firstPacket {
			firstPacket = false
			if pkt.SampleRate != 44100 {
				res.setRatio(float64(pkt.SampleRate) / 44100.0)
			}
		}

		if pkt.SampleRate != 44100 && !warnedRate {
			warnedRate = true
			log.info("stream sample rate is %d Hz; resampling to 44100 Hz for Lyrion", pkt.SampleRate)
		}

		if pkt.Channels > 2 && !warnedChannels {
			warnedChannels = true
			log.info("stream has %d channels, using channels 1+2 and ignoring the rest", pkt.Channels)
		}

		if haveFrame {
			switch {
			case pkt.Frame > lastFrame && pkt.Frame-lastFrame < 100000:
				st.gaps += int(pkt.Frame - lastFrame - 1)
			case pkt.Frame < lastFrame:
				// sender restarted, frame counter reset
			}
		}
		lastFrame = pkt.Frame
		haveFrame = true

		pcm, peak, err := toStereoS16(pkt, opts.channelMap)
		if err != nil {
			st.rejected++
			log.error("%v", err)
			continue
		}
		if peak > st.peak {
			st.peak = peak
		}

		// Measure the real frame rate; the VBAN header rate can be wrong (for
		// example VoiceMeeter sends its engine rate but labels it 44100).
		// Use the VBAN header rate as the stable reference. Only correct it when
		// the measured rate deviates clearly (some senders label the wrong
		// rate); frequent small corrections would make the pitch wobble.
		measureFrames += int64(pkt.SamplesPerFrame)
		if elapsed := time.Since(measureStart); elapsed >= time.Second {
			rate := float64(measureFrames) / elapsed.Seconds()
			st.inRate = rate
			expected := res.ratioValue() * 44100.0
			if expected > 0 && math.Abs(rate-expected)/expected > 0.05 {
				ratio := rate / 44100.0
				smoothed := res.ratioValue()*0.8 + ratio*0.2
				if math.Abs(smoothed-1) < 0.005 {
					smoothed = 1
				}
				res.setRatio(smoothed)
				log.info("measured %.0f Hz differs from the VBAN header; adjusting resample ratio to %.4f", rate, res.ratioValue())
			}
			measureFrames = 0
			measureStart = time.Now()

			if statsPath := statsFilePath(stream.Stream); statsPath != "" {
				_ = os.WriteFile(statsPath, []byte(fmt.Sprintf("inputRate=%.1f\nratio=%.4f\ndatagrams=%d\nframes=%d\nduration=%.2f\n", rate, res.ratioValue(), st.datagrams, st.frames, time.Since(st.start).Seconds())), 0644)
			}
		}

		outPCM := res.push(pcm, make([]byte, 0, len(pcm)+32))

		written := len(outPCM)
		werr := aw.write(outPCM)
		if werr != nil {
			if opts.device == "" && isClosedPipe(werr) {
				log.debug("output closed by consumer: %v", werr)
				st.consumerGone = true
				st.duration = time.Since(st.start)
				return st, nil
			}
			return st, fmt.Errorf("writing PCM output: %w", werr)
		}

		st.packets++
		st.bytesIn += int64(len(pkt.Payload))
		st.bytesOut += int64(written)
		st.frames += int64(pkt.SamplesPerFrame)
		lastWrite = time.Now()
	}

	st.duration = time.Since(st.start)
	return st, nil
}

func parseMulticastHost(host string) (net.IP, bool, error) {
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsMulticast() {
		return nil, false, nil
	}
	if ip.To4() == nil {
		return nil, true, fmt.Errorf("IPv6 multicast is not supported: %s", host)
	}
	return ip.To4(), true, nil
}

func listenWithRetry(ctx context.Context, stream streamURL, opts options, log *logger) (*net.UDPConn, error) {
	deadline := time.Now().Add(10 * time.Second)
	attempt := 0

	for {
		conn, err := listen(stream.Port, stream.Host, opts.iface, log)
		if err == nil {
			return conn, nil
		}

		attempt++
		if attempt == 1 || attempt%10 == 0 {
			log.info("cannot listen on UDP port %d yet (%v); retrying", stream.Port, err)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("cannot listen on UDP port %d: %w", stream.Port, err)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func listen(port int, host string, iface net.IP, log *logger) (*net.UDPConn, error) {
	group, multicast, err := parseMulticastHost(host)
	if err != nil {
		return nil, err
	}

	if multicast {
		// Multicast: bind 0.0.0.0 with SO_REUSEADDR and join the group. Several
		// receivers on this host can then bind the same port and each receives
		// its own copy of every datagram.
		lc := net.ListenConfig{Control: reuseAddrControl}
		pc, err := lc.ListenPacket(context.Background(), "udp4", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
		if err != nil {
			return nil, err
		}
		conn, ok := pc.(*net.UDPConn)
		if !ok {
			pc.Close()
			return nil, errors.New("unexpected packet connection type")
		}
		if err := joinMulticastGroup(conn, group, iface); err != nil {
			conn.Close()
			return nil, fmt.Errorf("cannot join multicast group %s: %w", host, err)
		}
		if err := conn.SetReadBuffer(4 << 20); err != nil {
			log.debug("could not enlarge UDP receive buffer: %v", err)
		}
		log.info("joined multicast group %s on UDP %d", host, port)
		return conn, nil
	}

	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: port})
	if err != nil {
		return nil, err
	}
	if err := conn.SetReadBuffer(4 << 20); err != nil {
		log.debug("could not enlarge UDP receive buffer: %v", err)
	}
	return conn, nil
}

func statsFilePath(stream string) string {
	if st, err := os.Stat(`C:\ProgramData\Lyrion`); err == nil && st.IsDir() {
		return filepath.Join(`C:\ProgramData\Lyrion`, "vban-stats-"+safeFileName(stream)+".txt")
	}
	return ""
}

func safeFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "Stream1"
	}
	return b.String()
}

func writeTitle(path, text string) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, []byte(text), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sourceAllowed(ip net.IP, allowed []net.IP) bool {
	for _, candidate := range allowed {
		if candidate.Equal(ip) {
			return true
		}
	}
	return false
}

func printStats(w io.Writer, st stats) {
	seconds := st.duration.Seconds()
	if seconds <= 0 {
		seconds = 0.001
	}

	peakPercent := float64(st.peak) / 32768 * 100

	fmt.Fprintln(w, "--- vban stats ---")
	fmt.Fprintf(w, "duration    : %.2f s\n", seconds)
	fmt.Fprintf(w, "datagrams   : %d\n", st.datagrams)
	fmt.Fprintf(w, "packets     : %d\n", st.packets)
	fmt.Fprintf(w, "rejected    : %d\n", st.rejected)
	fmt.Fprintf(w, "other stream: %d\n", st.otherStream)
	fmt.Fprintf(w, "texts       : %d\n", st.texts)
	fmt.Fprintf(w, "blocked     : %d\n", st.blocked)
	fmt.Fprintf(w, "gaps        : %d\n", st.gaps)
	fmt.Fprintf(w, "payload in  : %d bytes\n", st.bytesIn)
	fmt.Fprintf(w, "pcm out     : %d bytes\n", st.bytesOut)
	fmt.Fprintf(w, "silence out : %d bytes\n", st.silence)
	fmt.Fprintf(w, "frames      : %d (%.2f s at 44100 Hz)\n", st.frames, float64(st.frames)/44100)
	fmt.Fprintf(w, "peak        : %d (%.1f%%)\n", st.peak, peakPercent)
	if st.sampleRate != 0 {
		fmt.Fprintf(w, "format      : %d Hz / %s / %d ch\n", st.sampleRate, st.bitLabel, st.channels)
	}
	if st.inRate > 0 {
		fmt.Fprintf(w, "input rate  : %.1f Hz (resampled to 44100)\n", st.inRate)
	}
	if st.consumerGone {
		fmt.Fprintln(w, "note        : output closed by consumer")
	}
}
