package main

import (
	"encoding/binary"
	"testing"
)

func buildTextPacket(stream, text string) []byte {
	buf := make([]byte, vbanHeaderSize+len(text))
	copy(buf[0:4], "VBAN")
	buf[4] = protocolText
	buf[5] = 0
	buf[6] = 0
	buf[7] = 1 // UTF8
	name := make([]byte, vbanStreamNameSize)
	copy(name, stream)
	copy(buf[8:24], name)
	binary.LittleEndian.PutUint32(buf[24:28], 0)
	copy(buf[vbanHeaderSize:], text)
	return buf
}

func TestParseTextPacket(t *testing.T) {
	raw := buildTextPacket("Stream1", "Valaida Snow - Coconut Head")

	stream, text, ok := parseTextPacket(raw)
	if !ok {
		t.Fatalf("parseTextPacket returned ok=false")
	}
	if stream != "Stream1" {
		t.Fatalf("stream = %q, want Stream1", stream)
	}
	if text != "Valaida Snow - Coconut Head" {
		t.Fatalf("text = %q", text)
	}
}

func TestParseTextPacketNulTerminated(t *testing.T) {
	raw := buildTextPacket("Stream1", "Title\x00rest")
	_, text, ok := parseTextPacket(raw)
	if !ok || text != "Title" {
		t.Fatalf("got ok=%v text=%q, want Title", ok, text)
	}
}

func TestParseTextPacketRejectsAudio(t *testing.T) {
	audio := buildPacket("Stream1", 44100, 1, 2, 4, 0, payload16(4, 2))
	if _, _, ok := parseTextPacket(audio); ok {
		t.Fatalf("audio packet parsed as text")
	}
}

func TestSafeFileName(t *testing.T) {
	cases := map[string]string{
		"Stream1":     "Stream1",
		"My Stream/1": "My_Stream_1",
		"":            "Stream1",
		"a.b_c-1":     "a.b_c-1",
	}
	for in, want := range cases {
		if got := safeFileName(in); got != want {
			t.Fatalf("safeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}
