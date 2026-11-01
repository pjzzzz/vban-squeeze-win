package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func makePCM(frames int, value int16) []byte {
	b := make([]byte, frames*4)
	for i := 0; i < frames; i++ {
		binary.LittleEndian.PutUint16(b[i*4:], uint16(value))
		binary.LittleEndian.PutUint16(b[i*4+2:], uint16(value))
	}
	return b
}

func TestResamplerPassthrough(t *testing.T) {
	r := newResampler(1.0)
	var out []byte
	for i := 0; i < 10; i++ {
		out = r.push(makePCM(256, 1000), out)
	}
	// Linear interpolation with ratio 1 needs one extra frame at the end.
	if got, want := len(out)/4, 2560; math.Abs(float64(got-want)) > 1 {
		t.Fatalf("passthrough frames = %d, want %d", got, want)
	}
	if v := int16(binary.LittleEndian.Uint16(out[0:])); v != 1000 {
		t.Fatalf("first sample = %d, want 1000", v)
	}
}

func TestResampler48kTo44k(t *testing.T) {
	const framesIn = 48000
	ratio := 48000.0 / 44100.0

	r := newResampler(ratio)
	var out []byte

	sent := 0
	for sent < framesIn {
		n := 256
		if framesIn-sent < n {
			n = framesIn - sent
		}
		out = r.push(makePCM(n, 500), out)
		sent += n
	}

	frames := len(out) / 4
	want := int(float64(framesIn) / ratio)
	if math.Abs(float64(frames-want)) > 2 {
		t.Fatalf("resampled frames = %d, want about %d", frames, want)
	}
}

func TestResamplerClampsRatio(t *testing.T) {
	r := newResampler(10)
	if r.ratioValue() != 2.0 {
		t.Fatalf("ratio = %v, want 2.0", r.ratioValue())
	}
	r.setRatio(0.1)
	if r.ratioValue() != 0.5 {
		t.Fatalf("ratio = %v, want 0.5", r.ratioValue())
	}
}
