// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
//
// PCM conversion: VBAN carries little-endian samples; Lyrion's "pcm" stream
// format is 16-bit signed stereo. Everything is normalised to that format.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

// toStereoS16 converts a packet payload to 16-bit signed stereo PCM.
// channelMap holds 1-based channel indexes to use for left and right; when it
// is empty the natural mapping is used (mono is duplicated, >2 channels uses
// the first two).
func toStereoS16(p packet, channelMap []int) ([]byte, int, error) {
	out := make([]byte, p.SamplesPerFrame*4)
	peak := 0

	left, right, err := channelIndexes(p.Channels, channelMap)
	if err != nil {
		return nil, 0, err
	}

	step := p.Channels * p.BytesPerSample

	for f := 0; f < p.SamplesPerFrame; f++ {
		base := f * step

		l := int16(0)
		if left >= 0 && left < p.Channels {
			l = decodeSample(p.Payload[base+left*p.BytesPerSample:], p.BitFmt)
		}

		r := l
		if right >= 0 && right < p.Channels && right != left {
			r = decodeSample(p.Payload[base+right*p.BytesPerSample:], p.BitFmt)
		} else if right >= p.Channels || right < 0 {
			r = 0
		}

		binary.LittleEndian.PutUint16(out[f*4:], uint16(l))
		binary.LittleEndian.PutUint16(out[f*4+2:], uint16(r))

		if v := absInt16(l); v > peak {
			peak = v
		}
		if v := absInt16(r); v > peak {
			peak = v
		}
	}

	return out, peak, nil
}

func channelIndexes(channels int, channelMap []int) (int, int, error) {
	if len(channelMap) == 0 {
		if channels == 1 {
			return 0, 0, nil
		}
		return 0, 1, nil
	}

	left := channelMap[0] - 1
	right := left
	if len(channelMap) > 1 {
		right = channelMap[1] - 1
	}
	if left < 0 {
		return 0, 0, fmt.Errorf("channel indexes are 1-based, got %d", channelMap[0])
	}
	return left, right, nil
}

func decodeSample(b []byte, bitFmt int) int16 {
	switch bitFmt {
	case 0: // 8-bit unsigned
		return int16((int(b[0]) - 128) << 8)
	case 1: // 16-bit signed
		return int16(binary.LittleEndian.Uint16(b))
	case 2: // 24-bit signed
		v := int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16
		if v&0x800000 != 0 {
			v |= ^0xFFFFFF
		}
		return int16(v >> 8)
	case 3: // 32-bit signed
		return int16(int32(binary.LittleEndian.Uint32(b)) >> 16)
	case 4: // 32-bit float
		return floatToS16(float64(math.Float32frombits(binary.LittleEndian.Uint32(b))))
	case 5: // 64-bit float
		return floatToS16(math.Float64frombits(binary.LittleEndian.Uint64(b)))
	}
	return 0
}

func floatToS16(f float64) int16 {
	if f > 1 {
		f = 1
	} else if f < -1 {
		f = -1
	}
	return int16(math.Round(f * 32767))
}

func absInt16(v int16) int {
	if v < 0 {
		return -int(v)
	}
	return int(v)
}

// parseChannelMap parses the upstream-style "-c 1,2" channel list.
func parseChannelMap(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}

	var out []int
	cur := 0
	haveDigit := false

	flush := func() error {
		if !haveDigit {
			return fmt.Errorf("invalid channel list %q", s)
		}
		if cur < 1 || cur > 256 {
			return fmt.Errorf("channel %d out of range (1-256)", cur)
		}
		out = append(out, cur)
		cur = 0
		haveDigit = false
		return nil
	}

	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			cur = cur*10 + int(r-'0')
			haveDigit = true
		case r == ',':
			if err := flush(); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid character %q in channel list", r)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out, nil
}
