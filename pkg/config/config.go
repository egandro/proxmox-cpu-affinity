package config

import (
	"os"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	// CPUInfo defaults
	// DefaultRounds is set to 10 to average out noise from the OS scheduler and
	// background processes. This provides a stable latency measurement without
	// taking too long to initialize.
	DefaultRounds = 10
	// DefaultIterations is set to 100,000 to ensure the measurement duration
	// (~20ms) is long enough to overcome timer resolution limits and trigger
	// CPU frequency scaling, while keeping startup time reasonable.
	// Future improvement: This can be made dynamic based on the number of cores.
	DefaultIterations = 100_000

	// Service defaults
	DefaultServicePort = 8245
	DefaultServiceHost = "127.0.0.1"

	// Logging defaults
	DefaultLogDir      = "/var/log"
	DefaultLogFilename = "proxmox-cpu-affinity.log"
	DefaultLogFile     = DefaultLogDir + "/" + DefaultLogFilename

	// logger
	DefaultLogLevel = "info"

	// Proxmox defaults
	DefaultQemuServerPidDir = "/var/run/qemu-server"
	DefaultConfigFilename   = "/etc/default/proxmox-cpu-affinity"
)

// AdaptiveCpuInfoParameters calculates measurement parameters based on CPU count.
// Larger systems use reduced parameters to avoid excessive benchmark time.
// Returns (rounds, iterations).
func AdaptiveCpuInfoParameters() (int, int) {
	cpuCount := runtime.NumCPU()

	limits := []struct {
		cores      int
		rounds     int
		iterations int
	}{
		{16, 10, 100_000}, // Small systems: full precision
		{32, 5, 50_000},   // Medium systems: ~4x faster
		{64, 3, 25_000},   // Large systems: ~16x faster
		{128, 2, 10_000},  // Very large systems: ~50x faster
	}
	for _, l := range limits {
		if cpuCount <= l.cores {
			return l.rounds, l.iterations
		}
	}

	// Massive systems (>128 cores): ~100x faster
	return 2, 5_000
}

type Config struct {
	ServiceHost string
	ServicePort int
	LogLevel    string
	LogFile     string
	Rounds      int
	Iterations  int
}

func Load(filename string) *Config {
	if filename == "" {
		filename = DefaultConfigFilename
	}
	_ = godotenv.Load(filename)

	// Get adaptive defaults based on CPU count, allowing user overrides
	defaultRounds, defaultIterations := AdaptiveCpuInfoParameters()

	return &Config{
		ServiceHost: getEnv("PCA_HOST", DefaultServiceHost),
		ServicePort: getEnvInt("PCA_PORT", DefaultServicePort),
		LogLevel:    getEnv("PCA_LOG_LEVEL", DefaultLogLevel),
		LogFile:     getEnv("PCA_LOG_FILE", DefaultLogFile),
		Rounds:      getEnvInt("PCA_ROUNDS", defaultRounds),
		Iterations:  getEnvInt("PCA_ITERATIONS", defaultIterations),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}
