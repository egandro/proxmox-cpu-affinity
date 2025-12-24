package config

import (
	"os"
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

	return &Config{
		ServiceHost: getEnv("PCA_HOST", DefaultServiceHost),
		ServicePort: getEnvInt("PCA_PORT", DefaultServicePort),
		LogLevel:    getEnv("PCA_LOG_LEVEL", DefaultLogLevel),
		LogFile:     getEnv("PCA_LOG_FILE", DefaultLogFile),
		Rounds:      getEnvInt("PCA_ROUNDS", DefaultRounds),
		Iterations:  getEnvInt("PCA_ITERATIONS", DefaultIterations),
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
