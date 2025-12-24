package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdaptiveCpuInfoParameters(t *testing.T) {
	rounds, iterations := AdaptiveCpuInfoParameters()

	// Verify we get valid positive values
	assert.Greater(t, rounds, 0, "rounds should be positive")
	assert.Greater(t, iterations, 0, "iterations should be positive")
}

func TestAdaptiveCpuInfoParameters_DeveloperHack(t *testing.T) {
	t.Skip("Skipping developer adaptive cpuinfo parameters test.")

	rounds, iterations := AdaptiveCpuInfoParameters()
	cpuCount := runtime.NumCPU()

	// Verify we get valid positive values
	assert.Greater(t, rounds, 0, "rounds should be positive")
	assert.Greater(t, iterations, 0, "iterations should be positive")

	// Verify the scaling logic based on CPU count
	switch {
	case cpuCount <= 16:
		assert.Equal(t, 10, rounds)
		assert.Equal(t, 100_000, iterations)
	case cpuCount <= 32:
		assert.Equal(t, 5, rounds)
		assert.Equal(t, 50_000, iterations)
	case cpuCount <= 64:
		assert.Equal(t, 3, rounds)
		assert.Equal(t, 25_000, iterations)
	case cpuCount <= 128:
		assert.Equal(t, 2, rounds)
		assert.Equal(t, 10_000, iterations)
	default:
		assert.Equal(t, 2, rounds)
		assert.Equal(t, 5_000, iterations)
	}
}

func TestLoad(t *testing.T) {
	// Test loading with defaults (no env vars set)
	cfg := Load("")

	assert.Equal(t, DefaultServiceHost, cfg.ServiceHost)
	assert.Equal(t, DefaultServicePort, cfg.ServicePort)
	assert.Equal(t, DefaultLogLevel, cfg.LogLevel)
	assert.Equal(t, DefaultLogFile, cfg.LogFile)

	// Rounds and iterations should match adaptive defaults
	expectedRounds, expectedIterations := AdaptiveCpuInfoParameters()
	assert.Equal(t, expectedRounds, cfg.Rounds)
	assert.Equal(t, expectedIterations, cfg.Iterations)
}

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
