// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
//
// The receiver outputs 16-bit stereo PCM at the rate passed with
// --target-rate (the rate Lyrion announces to the player). VBAN senders do not
// always use that rate (VoiceMeeter uses its engine rate, and the VBAN header
// rate can even be wrong), so the stream is resampled with linear
// interpolation based on the measured frame rate.
package main

import "encoding/binary"

type frame struct {
	l int16
	r int16
}

type resampler struct {
	buf   []frame
	phase float64
	ratio float64 // input frames per output frame
}

func newResampler(ratio float64) *resampler {
	r := &resampler{}
	r.setRatio(ratio)
	return r
}

func (r *resampler) setRatio(ratio float64) {
	if ratio < 0.5 {
		ratio = 0.5
	} else if ratio > 2.0 {
		ratio = 2.0
	}
	r.ratio = ratio
}

func (r *resampler) ratioValue() float64 {
	return r.ratio
}

// push appends 16-bit stereo PCM and returns the resampled PCM.
func (r *resampler) push(pcm []byte, out []byte) []byte {
	for i := 0; i+3 < len(pcm); i += 4 {
		r.buf = append(r.buf, frame{
			l: int16(binary.LittleEndian.Uint16(pcm[i:])),
			r: int16(binary.LittleEndian.Uint16(pcm[i+2:])),
		})
	}

	for r.phase+1 < float64(len(r.buf)) {
		i := int(r.phase)
		frac := r.phase - float64(i)
		a := r.buf[i]
		b := r.buf[i+1]

		l := int16(float64(a.l)*(1-frac) + float64(b.l)*frac)
		rr := int16(float64(a.r)*(1-frac) + float64(b.r)*frac)

		var b4 [4]byte
		binary.LittleEndian.PutUint16(b4[0:], uint16(l))
		binary.LittleEndian.PutUint16(b4[2:], uint16(rr))
		out = append(out, b4[:]...)

		r.phase += r.ratio
	}

	// Drop the frames that are fully consumed and keep the fractional phase.
	if drop := int(r.phase); drop > 0 {
		if drop > len(r.buf) {
			drop = len(r.buf)
		}
		r.buf = r.buf[drop:]
		r.phase -= float64(drop)
	}

	// Safety valve: never let the buffer grow without bound if the measured
	// rate is wrong for a while.
	if len(r.buf) > 16384 {
		r.buf = r.buf[len(r.buf)-8192:]
	}

	return out
}
