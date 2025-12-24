package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdaptiveParameters(t *testing.T) {
	rounds, iterations := AdaptiveParameters()
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
	expectedRounds, expectedIterations := AdaptiveParameters()
	assert.Equal(t, expectedRounds, cfg.Rounds)
	assert.Equal(t, expectedIterations, cfg.Iterations)
}
