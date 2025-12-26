package config

import (
	"os"
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
	cpuCount := numCPU()

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

	assert.Equal(t, DefaultLogLevel, cfg.LogLevel)
	assert.Equal(t, ConstantLogFile, cfg.LogFile)
	assert.Equal(t, ConstantSocketFile, cfg.SocketFile)
	assert.Equal(t, DefaultSocketRetry, cfg.SocketRetry)
	assert.Equal(t, DefaultSocketSleep, cfg.SocketSleep)
	assert.Equal(t, DefaultSocketTimeout, cfg.SocketTimeout)
	assert.Equal(t, DefaultSocketPingOnPreStart, cfg.SocketPingOnPreStart)

	// Rounds and iterations should match adaptive defaults
	expectedRounds, expectedIterations := AdaptiveCpuInfoParameters()
	assert.Equal(t, expectedRounds, cfg.Rounds)
	assert.Equal(t, expectedIterations, cfg.Iterations)
}

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		fallback bool
		expected bool
	}{
		{"True value", "true", true, false, true},
		{"False value", "false", true, true, false},
		{"1 value", "1", true, false, true},
		{"0 value", "0", true, true, false},
		{"Invalid value uses fallback", "invalid", true, true, true},
		{"Empty value uses fallback", "", true, false, false},
		{"Unset uses fallback true", "", false, true, true},
		{"Unset uses fallback false", "", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_BOOL_VAR"
			_ = os.Unsetenv(key)
			if tt.setEnv {
				_ = os.Setenv(key, tt.envValue)
				defer func() { _ = os.Unsetenv(key) }()
			}
			result := getEnvBool(key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		setEnv   bool
		fallback int
		expected int
	}{
		{"Valid int", "42", true, 0, 42},
		{"Negative int", "-10", true, 0, -10},
		{"Zero", "0", true, 100, 0},
		{"Invalid uses fallback", "not-a-number", true, 99, 99},
		{"Float uses fallback", "3.14", true, 5, 5},
		{"Empty uses fallback", "", true, 7, 7},
		{"Unset uses fallback", "", false, 123, 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT_VAR"
			_ = os.Unsetenv(key)
			if tt.setEnv {
				_ = os.Setenv(key, tt.envValue)
				defer func() { _ = os.Unsetenv(key) }()
			}
			result := getEnvInt(key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Save current env and restore after test
	envVars := []string{
		"PCA_LOG_LEVEL", "PCA_LOG_FILE", "PCA_SOCKET_FILE", "PCA_ROUNDS", "PCA_ITERATIONS",
		"PCA_SOCKET_RETRY", "PCA_SOCKET_SLEEP", "PCA_SOCKET_TIMEOUT",
		"PCA_SOCKET_PING_ON_PRESTART", "PCA_CPU_HOTPLUG_WATCHDOG",
	}
	savedEnv := make(map[string]string)
	for _, key := range envVars {
		savedEnv[key], _ = os.LookupEnv(key)
		_ = os.Unsetenv(key)
	}
	defer func() {
		for key, val := range savedEnv {
			if val != "" {
				_ = os.Setenv(key, val)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}()

	// Set custom values
	_ = os.Setenv("PCA_LOG_LEVEL", "debug")
	_ = os.Setenv("PCA_LOG_FILE", "/tmp/test.log")
	_ = os.Setenv("PCA_SOCKET_FILE", "/tmp/test.sock")
	_ = os.Setenv("PCA_ROUNDS", "5")
	_ = os.Setenv("PCA_ITERATIONS", "50000")
	_ = os.Setenv("PCA_SOCKET_RETRY", "3")
	_ = os.Setenv("PCA_SOCKET_SLEEP", "5")
	_ = os.Setenv("PCA_SOCKET_TIMEOUT", "60")
	_ = os.Setenv("PCA_SOCKET_PING_ON_PRESTART", "false")
	_ = os.Setenv("PCA_CPU_HOTPLUG_WATCHDOG", "false")

	cfg := Load("")

	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "/tmp/test.log", cfg.LogFile)
	assert.Equal(t, "/tmp/test.sock", cfg.SocketFile)
	assert.Equal(t, 5, cfg.Rounds)
	assert.Equal(t, 50000, cfg.Iterations)
	assert.Equal(t, 3, cfg.SocketRetry)
	assert.Equal(t, 5, cfg.SocketSleep)
	assert.Equal(t, 60, cfg.SocketTimeout)
	assert.False(t, cfg.SocketPingOnPreStart)
	assert.False(t, cfg.CPUHotplugWatchdog)
}

func TestGetEnv(t *testing.T) {
	key := "TEST_STRING_VAR"
	_ = os.Unsetenv(key)

	// Test fallback when not set
	result := getEnv(key, "fallback_value")
	assert.Equal(t, "fallback_value", result)

	// Test when set
	_ = os.Setenv(key, "actual_value")
	defer func() { _ = os.Unsetenv(key) }()
	result = getEnv(key, "fallback_value")
	assert.Equal(t, "actual_value", result)
}
