// vban_url_receptor - Windows VBAN receiver for the Lyrion Music Server plugin.
//
// VBAN protocol constants and header layout follow quiniouben/vban
// (src/vban/vban.h, src/common/packet.c), GPL-3.0.
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	vbanHeaderSize    = 28 // 4 magic + 4 format + 16 streamname + 4 frame counter
	vbanMaxPacketSize = 1464

	protocolMask  = 0xE0
	protocolAudio = 0x00

	codecMask = 0xF0
	codecPCM  = 0x00

	srMask      = 0x1F
	bitFmtMask  = 0x07
	reservedBit = 0x08
)

// sampleRates maps the 5-bit VBAN sample rate index to Hz.
var sampleRates = []int{
	6000, 12000, 24000, 48000, 96000, 192000, 384000,
	8000, 16000, 32000, 64000, 128000, 256000, 512000,
	11025, 22050, 44100, 88200, 176400, 352800, 705600,
}

// bytesPerSample maps the 3-bit bit format index to the number of bytes per
// sample. 12-bit (6) and 10-bit (7) are not supported by this receiver.
var bytesPerSample = map[int]int{
	0: 1, // 8-bit int
	1: 2, // 16-bit int
	2: 3, // 24-bit int
	3: 4, // 32-bit int
	4: 4, // 32-bit float
	5: 8, // 64-bit float
}

var (
	errShortPacket       = errors.New("packet too short")
	errBadMagic          = errors.New("invalid VBAN magic")
	errNotAudio          = errors.New("not an audio sub-protocol packet")
	errNotPCM            = errors.New("not a PCM packet")
	errReservedBit       = errors.New("reserved format bit is set")
	errBadSampleRate     = errors.New("unknown sample rate index")
	errUnsupportedBitFmt = errors.New("unsupported bit format (12/10-bit)")
)

// packet is a validated VBAN audio packet.
type packet struct {
	SampleRate      int
	BitFmt          int
	BytesPerSample  int
	Channels        int
	SamplesPerFrame int
	StreamName      string
	Frame           uint32
	Payload         []byte
}

func (p packet) bitDepthLabel() string {
	switch p.BitFmt {
	case 0:
		return "8-bit"
	case 1:
		return "16-bit"
	case 2:
		return "24-bit"
	case 3:
		return "32-bit int"
	case 4:
		return "32-bit float"
	case 5:
		return "64-bit float"
	}
	return "unknown"
}

// parsePacket validates a raw datagram and returns the decoded header.
func parsePacket(b []byte) (packet, error) {
	var p packet

	if len(b) < vbanHeaderSize+1 {
		return p, errShortPacket
	}

	if string(b[0:4]) != "VBAN" {
		return p, errBadMagic
	}

	if b[4]&protocolMask != protocolAudio {
		return p, errNotAudio
	}

	srIndex := int(b[4] & srMask)
	if srIndex >= len(sampleRates) {
		return p, errBadSampleRate
	}

	if b[7]&reservedBit != 0 {
		return p, errReservedBit
	}

	if b[7]&codecMask != codecPCM {
		return p, errNotPCM
	}

	bitFmt := int(b[7] & bitFmtMask)
	size, ok := bytesPerSample[bitFmt]
	if !ok {
		return p, errUnsupportedBitFmt
	}

	p.SampleRate = sampleRates[srIndex]
	p.BitFmt = bitFmt
	p.BytesPerSample = size
	p.SamplesPerFrame = int(b[5]) + 1
	p.Channels = int(b[6]) + 1
	p.Frame = binary.LittleEndian.Uint32(b[24:28])

	name := b[8:24]
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	p.StreamName = string(name)

	p.Payload = b[vbanHeaderSize:]

	expected := p.SamplesPerFrame * p.Channels * p.BytesPerSample
	if len(p.Payload) != expected {
		return p, fmt.Errorf("payload is %d bytes, expected %d", len(p.Payload), expected)
	}

	return p, nil
}

// buildPacket is used by tests and tools to create a VBAN audio datagram.
func buildPacket(stream string, sampleRate int, bitFmt int, channels int, samplesPerFrame int, frame uint32, payload []byte) []byte {
	buf := make([]byte, vbanHeaderSize+len(payload))

	copy(buf[0:4], "VBAN")

	srIndex := 0
	for i, v := range sampleRates {
		if v == sampleRate {
			srIndex = i
			break
		}
	}

	buf[4] = byte(srIndex) | protocolAudio
	buf[5] = byte(samplesPerFrame - 1)
	buf[6] = byte(channels - 1)
	buf[7] = byte(bitFmt)

	name := make([]byte, vbanStreamNameSize)
	copy(name, stream)
	copy(buf[8:24], name)

	binary.LittleEndian.PutUint32(buf[24:28], frame)
	copy(buf[vbanHeaderSize:], payload)

	return buf
}
