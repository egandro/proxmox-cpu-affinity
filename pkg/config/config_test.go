package config

import (
	"os"
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Localhost binding - no error",
			config: Config{
				ServiceHost: "127.0.0.1",
			},
			expectError: false,
		},
		{
			name: "Empty host (localhost) - no error",
			config: Config{
				ServiceHost: "",
			},
			expectError: false,
		},
		{
			name: "Non-localhost without flag - error",
			config: Config{
				ServiceHost:         "0.0.0.0",
				InsecureAllowRemote: false,
			},
			expectError: true,
		},
		{
			name: "Non-localhost with flag - no error (warning printed)",
			config: Config{
				ServiceHost:         "0.0.0.0",
				InsecureAllowRemote: true,
			},
			expectError: false,
		},
		{
			name: "External IP without flag - error",
			config: Config{
				ServiceHost:         "192.168.1.100",
				InsecureAllowRemote: false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "non-localhost")
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
			os.Unsetenv(key)
			if tt.setEnv {
				os.Setenv(key, tt.envValue)
				defer os.Unsetenv(key)
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
			os.Unsetenv(key)
			if tt.setEnv {
				os.Setenv(key, tt.envValue)
				defer os.Unsetenv(key)
			}
			result := getEnvInt(key, tt.fallback)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	// Save current env and restore after test
	envVars := []string{
		"PCA_HOST", "PCA_PORT", "PCA_INSECURE_ALLOW_REMOTE",
		"PCA_LOG_LEVEL", "PCA_LOG_FILE", "PCA_ROUNDS", "PCA_ITERATIONS",
		"PCA_WEBHOOK_RETRY", "PCA_WEBHOOK_SLEEP", "PCA_WEBHOOK_TIMEOUT",
		"PCA_WEBHOOK_PING_ON_PRESTART",
	}
	savedEnv := make(map[string]string)
	for _, key := range envVars {
		savedEnv[key], _ = os.LookupEnv(key)
		os.Unsetenv(key)
	}
	defer func() {
		for key, val := range savedEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Set custom values
	os.Setenv("PCA_HOST", "0.0.0.0")
	os.Setenv("PCA_PORT", "9999")
	os.Setenv("PCA_INSECURE_ALLOW_REMOTE", "true")
	os.Setenv("PCA_LOG_LEVEL", "debug")
	os.Setenv("PCA_LOG_FILE", "/tmp/test.log")
	os.Setenv("PCA_ROUNDS", "5")
	os.Setenv("PCA_ITERATIONS", "50000")
	os.Setenv("PCA_WEBHOOK_RETRY", "3")
	os.Setenv("PCA_WEBHOOK_SLEEP", "5")
	os.Setenv("PCA_WEBHOOK_TIMEOUT", "60")
	os.Setenv("PCA_WEBHOOK_PING_ON_PRESTART", "false")

	cfg := Load("")

	assert.Equal(t, "0.0.0.0", cfg.ServiceHost)
	assert.Equal(t, 9999, cfg.ServicePort)
	assert.True(t, cfg.InsecureAllowRemote)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "/tmp/test.log", cfg.LogFile)
	assert.Equal(t, 5, cfg.Rounds)
	assert.Equal(t, 50000, cfg.Iterations)
	assert.Equal(t, 3, cfg.WebhookRetry)
	assert.Equal(t, 5, cfg.WebhookSleep)
	assert.Equal(t, 60, cfg.WebhookTimeout)
	assert.False(t, cfg.WebhookPingOnPreStart)
}

func TestGetEnv(t *testing.T) {
	key := "TEST_STRING_VAR"
	os.Unsetenv(key)

	// Test fallback when not set
	result := getEnv(key, "fallback_value")
	assert.Equal(t, "fallback_value", result)

	// Test when set
	os.Setenv(key, "actual_value")
	defer os.Unsetenv(key)
	result = getEnv(key, "fallback_value")
	assert.Equal(t, "actual_value", result)
}
