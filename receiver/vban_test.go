package main

import (
	"encoding/binary"
	"errors"
	"testing"
)

func payload16(frames int, channels int) []byte {
	b := make([]byte, frames*channels*2)
	for i := 0; i < frames*channels; i++ {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(int16(i)))
	}
	return b
}

func TestParsePacket(t *testing.T) {
	payload := payload16(8, 2)
	raw := buildPacket("Stream1", 44100, 1, 2, 8, 42, payload)

	p, err := parsePacket(raw)
	if err != nil {
		t.Fatalf("parsePacket returned error: %v", err)
	}

	if p.SampleRate != 44100 {
		t.Fatalf("SampleRate = %d, want 44100", p.SampleRate)
	}
	if p.BitFmt != 1 || p.BytesPerSample != 2 {
		t.Fatalf("bit format = %d/%d, want 1/2", p.BitFmt, p.BytesPerSample)
	}
	if p.Channels != 2 {
		t.Fatalf("Channels = %d, want 2", p.Channels)
	}
	if p.SamplesPerFrame != 8 {
		t.Fatalf("SamplesPerFrame = %d, want 8", p.SamplesPerFrame)
	}
	if p.StreamName != "Stream1" {
		t.Fatalf("StreamName = %q, want Stream1", p.StreamName)
	}
	if p.Frame != 42 {
		t.Fatalf("Frame = %d, want 42", p.Frame)
	}
	if len(p.Payload) != len(payload) {
		t.Fatalf("payload length = %d, want %d", len(p.Payload), len(payload))
	}
}

func TestParsePacketStreamNamePadding(t *testing.T) {
	raw := buildPacket("AB", 44100, 1, 2, 4, 0, payload16(4, 2))
	p, err := parsePacket(raw)
	if err != nil {
		t.Fatalf("parsePacket returned error: %v", err)
	}
	if p.StreamName != "AB" {
		t.Fatalf("StreamName = %q, want AB", p.StreamName)
	}
}

func TestParsePacketErrors(t *testing.T) {
	valid := buildPacket("Stream1", 44100, 1, 2, 4, 0, payload16(4, 2))

	mutate := func(fn func([]byte)) []byte {
		cp := append([]byte(nil), valid...)
		fn(cp)
		return cp
	}

	tests := []struct {
		name string
		raw  []byte
		want error
	}{
		{"short", valid[:10], errShortPacket},
		{"magic", mutate(func(b []byte) { b[0] = 'X' }), errBadMagic},
		{"not audio", mutate(func(b []byte) { b[4] = 0x40 | 0x10 }), errNotAudio},
		{"not pcm", mutate(func(b []byte) { b[7] = 0x10 | 1 }), errNotPCM},
		{"reserved bit", mutate(func(b []byte) { b[7] |= 0x08 }), errReservedBit},
		{"bad sample rate", mutate(func(b []byte) { b[4] = 0x1F }), errBadSampleRate},
		{"12 bit", mutate(func(b []byte) { b[7] = 6 }), errUnsupportedBitFmt},
		{"payload mismatch", valid[:len(valid)-2], nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parsePacket(tc.raw)
			if err == nil {
				t.Fatalf("parsePacket expected an error")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("parsePacket error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestParsePacketEmptyPayload(t *testing.T) {
	raw := buildPacket("Stream1", 44100, 1, 2, 4, 0, payload16(4, 2))
	if _, err := parsePacket(raw[:vbanHeaderSize]); err == nil {
		t.Fatalf("expected error for header-only packet")
	}
}
