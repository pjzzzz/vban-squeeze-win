package main

import "testing"

func TestParseMulticastHost(t *testing.T) {
	tests := []struct {
		host    string
		wantMC  bool
		wantErr bool
	}{
		{"239.255.42.42", true, false},
		{"224.0.0.1", true, false},
		{"192.168.178.30", false, false},
		{"127.0.0.1", false, false},
		{"example.local", false, false},
		{"", false, false},
		{"ff02::1", true, true},
	}

	for _, tc := range tests {
		ip, mc, err := parseMulticastHost(tc.host)
		if (err != nil) != tc.wantErr {
			t.Fatalf("parseMulticastHost(%q) error = %v, wantErr %v", tc.host, err, tc.wantErr)
		}
		if mc != tc.wantMC {
			t.Fatalf("parseMulticastHost(%q) multicast = %v, want %v", tc.host, mc, tc.wantMC)
		}
		if mc && err == nil && (ip == nil || ip.To4() == nil) {
			t.Fatalf("parseMulticastHost(%q) returned a non-IPv4 group", tc.host)
		}
	}
}
