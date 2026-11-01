package main

import (
	"strings"
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		host   string
		port   int
		stream string
	}{
		{"full", "vban://192.168.178.79/Stream1", "192.168.178.79", 6980, "Stream1"},
		{"with port", "vban://192.168.178.79:7000/Stream2", "192.168.178.79", 7000, "Stream2"},
		{"host only", "vban://192.168.178.79", "192.168.178.79", 6980, "Stream1"},
		{"empty url", "vban://", "127.0.0.1", 6980, "Stream1"},
		{"no scheme", "192.168.178.79/MyStream", "192.168.178.79", 6980, "MyStream"},
		{"ipv6", "vban://[::1]:6980/Stream1", "::1", 6980, "Stream1"},
		{"encoded stream", "vban://host/My%20Stream", "host", 6980, "My Stream"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseURL(tc.in)
			if err != nil {
				t.Fatalf("parseURL(%q) returned error: %v", tc.in, err)
			}
			if got.Host != tc.host || got.Port != tc.port || got.Stream != tc.stream {
				t.Fatalf("parseURL(%q) = %+v, want host=%q port=%d stream=%q",
					tc.in, got, tc.host, tc.port, tc.stream)
			}
		})
	}
}

func TestParseURLErrors(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"http://host/Stream1",
		"vban://host:0/Stream1",
		"vban://host:70000/Stream1",
		"vban://host/" + strings.Repeat("x", 17),
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			if _, err := parseURL(in); err == nil {
				t.Fatalf("parseURL(%q) expected an error", in)
			}
		})
	}
}

func TestStreamURLString(t *testing.T) {
	u := streamURL{Host: "10.0.0.2", Port: 6980, Stream: "Stream1"}
	if got, want := u.String(), "vban://10.0.0.2:6980/Stream1"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}
