package main

import (
	"encoding/binary"
	"math"
	"testing"
)

func makePacket(bitFmt, channels, frames int, payload []byte) packet {
	return packet{
		SampleRate:      44100,
		BitFmt:          bitFmt,
		BytesPerSample:  bytesPerSample[bitFmt],
		Channels:        channels,
		SamplesPerFrame: frames,
		Payload:         payload,
	}
}

func s16(b []byte, i int) int16 {
	return int16(binary.LittleEndian.Uint16(b[i*2:]))
}
func putS16(b []byte, sample int, v int16) {
	binary.LittleEndian.PutUint16(b[sample*2:], uint16(v))
}

func TestToStereoS16StereoPassthrough(t *testing.T) {
	payload := make([]byte, 2*2*2)
	putS16(payload, 0, 1000)
	putS16(payload, 1, -1000)
	putS16(payload, 2, 2000)
	putS16(payload, 3, -2000)

	out, peak, err := toStereoS16(makePacket(1, 2, 2, payload), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 8 {
		t.Fatalf("output length = %d, want 8", len(out))
	}
	if got := s16(out, 0); got != 1000 {
		t.Fatalf("sample 0 = %d, want 1000", got)
	}
	if got := s16(out, 1); got != -1000 {
		t.Fatalf("sample 1 = %d, want -1000", got)
	}
	if got := s16(out, 2); got != 2000 {
		t.Fatalf("sample 2 = %d, want 2000", got)
	}
	if got := s16(out, 3); got != -2000 {
		t.Fatalf("sample 3 = %d, want -2000", got)
	}
	if peak != 2000 {
		t.Fatalf("peak = %d, want 2000", peak)
	}
}

func TestToStereoS16MonoDuplicated(t *testing.T) {
	payload := make([]byte, 3*2)
	putS16(payload, 0, 1234)
	putS16(payload, 1, -1234)
	putS16(payload, 2, 0)

	out, _, err := toStereoS16(makePacket(1, 1, 3, payload), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < 3; i++ {
		l := s16(out, i*2)
		r := s16(out, i*2+1)
		if l != r {
			t.Fatalf("frame %d: left %d != right %d", i, l, r)
		}
	}
	if s16(out, 0) != 1234 {
		t.Fatalf("sample = %d, want 1234", s16(out, 0))
	}
}

func TestToStereoS16Multichannel(t *testing.T) {
	payload := make([]byte, 6*2)
	values := []int16{100, 200, 300, 400, 500, 600}
	for i, v := range values {
		binary.LittleEndian.PutUint16(payload[i*2:], uint16(v))
	}

	out, _, err := toStereoS16(makePacket(1, 6, 1, payload), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s16(out, 0) != 100 || s16(out, 1) != 200 {
		t.Fatalf("got %d/%d, want 100/200", s16(out, 0), s16(out, 1))
	}
}

func TestToStereoS16ChannelMap(t *testing.T) {
	payload := make([]byte, 2*2)
	putS16(payload, 0, 11)
	putS16(payload, 1, 22)

	out, _, err := toStereoS16(makePacket(1, 2, 1, payload), []int{2, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s16(out, 0) != 22 || s16(out, 1) != 11 {
		t.Fatalf("got %d/%d, want 22/11", s16(out, 0), s16(out, 1))
	}
}

func TestToStereoS16ChannelMapOutOfRange(t *testing.T) {
	payload := make([]byte, 2*2)
	putS16(payload, 0, 11)
	putS16(payload, 1, 22)

	out, _, err := toStereoS16(makePacket(1, 2, 1, payload), []int{3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s16(out, 0) != 0 || s16(out, 1) != 0 {
		t.Fatalf("got %d/%d, want silence", s16(out, 0), s16(out, 1))
	}
}

func TestToStereoS16BitFormats(t *testing.T) {
	t.Run("24 bit", func(t *testing.T) {
		payload := []byte{0x56, 0x34, 0x12, 0x00, 0xFF, 0xFF}
		out, _, err := toStereoS16(makePacket(2, 1, 2, payload), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s16(out, 0); got != 0x1234 {
			t.Fatalf("sample = %#x, want 0x1234", got)
		}
		if got := s16(out, 2); got != -1 {
			t.Fatalf("sample = %d, want -1", got)
		}
	})

	t.Run("32 bit int", func(t *testing.T) {
		payload := make([]byte, 4)
		binary.LittleEndian.PutUint32(payload, 0x12345678)
		out, _, err := toStereoS16(makePacket(3, 1, 1, payload), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s16(out, 0); got != 0x1234 {
			t.Fatalf("sample = %#x, want 0x1234", got)
		}
	})

	t.Run("32 bit float", func(t *testing.T) {
		payload := make([]byte, 4)
		binary.LittleEndian.PutUint32(payload, math.Float32bits(0.5))
		out, _, err := toStereoS16(makePacket(4, 1, 1, payload), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s16(out, 0); got != 16384 {
			t.Fatalf("sample = %d, want 16384", got)
		}
	})

	t.Run("8 bit unsigned", func(t *testing.T) {
		payload := []byte{128, 255, 0}
		out, _, err := toStereoS16(makePacket(0, 1, 3, payload), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s16(out, 0); got != 0 {
			t.Fatalf("sample = %d, want 0", got)
		}
		if got := s16(out, 2); got != 32512 {
			t.Fatalf("sample = %d, want 32512", got)
		}
		if got := s16(out, 4); got != -32768 {
			t.Fatalf("sample = %d, want -32768", got)
		}
	})
}

func TestParseChannelMap(t *testing.T) {
	tests := []struct {
		in   string
		want []int
	}{
		{"", nil},
		{"1", []int{1}},
		{"1,2", []int{1, 2}},
		{"2", []int{2}},
		{"1,2,3", []int{1, 2}},
	}

	for _, tc := range tests {
		got, err := parseChannelMap(tc.in)
		if err != nil {
			t.Fatalf("parseChannelMap(%q) error: %v", tc.in, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("parseChannelMap(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("parseChannelMap(%q) = %v, want %v", tc.in, got, tc.want)
			}
		}
	}

	for _, in := range []string{"0", "257", "a", "1,", ",1", "1;2"} {
		if _, err := parseChannelMap(in); err == nil {
			t.Fatalf("parseChannelMap(%q) expected an error", in)
		}
	}
}
