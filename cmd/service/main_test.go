package main

import "testing"

func TestIsLocalhostAddr(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		// Localhost addresses
		{"127.0.0.1", true},
		{"localhost", true},
		{"::1", true},
		{"", true},

		// Non-localhost addresses
		{"0.0.0.0", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isLocalhostAddr(tt.host)
			if got != tt.expected {
				t.Errorf("isLocalhostAddr(%q) = %v, want %v", tt.host, got, tt.expected)
			}
		})
	}
}
