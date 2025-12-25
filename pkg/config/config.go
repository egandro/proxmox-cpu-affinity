package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

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
	DefaultServicePort         = 8245
	DefaultServiceHost         = "127.0.0.1"
	DefaultInsecureAllowRemote = false

	// Logging defaults
	DefaultLogDir      = "/var/log"
	DefaultLogFilename = "proxmox-cpu-affinity.log"
	DefaultLogFile     = DefaultLogDir + "/" + DefaultLogFilename

	// logger
	DefaultLogLevel = "info"

	// Proxmox defaults
	DefaultQemuServerPidDir   = "/var/run/qemu-server"
	DefaultConfigFilename     = "/etc/default/proxmox-cpu-affinity"
	DefaultProxmoxQM          = "/usr/sbin/qm"
	DefaultProxmoxHaManager   = "/usr/sbin/ha-manager"
	DefaultProxmoxConfigDir   = "/etc/pve"
	DefaultHookScriptFilename = "proxmox-cpu-affinity-hook"

	// Webhook defaults
	DefaultWebhookRetry           = 10
	DefaultWebhookSleep           = 10 // in seconds
	DefaultWebhookTimeout         = 30 // in seconds
	DefaultWebhookPingOnPreStart  = true
	DefaultCPUHotplugWatchdog     = true
	MaxCalculationRankingDuration = 2 * time.Minute

	// CPUHotplugBatchWindow is the time window to group hotplug events.
	// When a CPU event occurs, we wait this long for subsequent events
	// to arrive (debouncing) before triggering a topology recalculation.
	CPUHotplugBatchWindow = 5 * time.Second
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
	ServiceHost           string
	ServicePort           int
	InsecureAllowRemote   bool
	LogLevel              string
	LogFile               string
	Rounds                int
	Iterations            int
	WebhookRetry          int
	WebhookSleep          int // in seconds
	WebhookPingOnPreStart bool
	WebhookTimeout        int // in seconds
	CPUHotplugWatchdog    bool
}

func (c *Config) Validate() error {
	if !isLocalhostAddr(c.ServiceHost) {
		if !c.InsecureAllowRemote {
			return fmt.Errorf(`binding to non-localhost address %q exposes an unauthenticated API.

This service has no authentication. Binding to a network-accessible address
allows any host on the network to trigger CPU affinity changes on your VMs.

If you understand the risks and want to proceed anyway, use:
    --insecure-allow-remote
    or set PCA_INSECURE_ALLOW_REMOTE=true`, c.ServiceHost)
		}
		fmt.Fprintf(os.Stderr, "WARNING: Binding to %q - unauthenticated API will be network-accessible!\n", c.ServiceHost)
	}
	return nil
}

func isLocalhostAddr(host string) bool {
	switch host {
	case "127.0.0.1", "localhost", "::1", "":
		return true
	}
	return false
}

func Load(filename string) *Config {
	if filename == "" {
		filename = DefaultConfigFilename
	}
	_ = godotenv.Load(filename)

	// Get adaptive defaults based on CPU count, allowing user overrides
	defaultRounds, defaultIterations := AdaptiveCpuInfoParameters()

	return &Config{
		ServiceHost:           getEnv("PCA_HOST", DefaultServiceHost),
		ServicePort:           getEnvInt("PCA_PORT", DefaultServicePort),
		InsecureAllowRemote:   getEnvBool("PCA_INSECURE_ALLOW_REMOTE", DefaultInsecureAllowRemote),
		LogLevel:              getEnv("PCA_LOG_LEVEL", DefaultLogLevel),
		LogFile:               getEnv("PCA_LOG_FILE", DefaultLogFile),
		Rounds:                getEnvInt("PCA_ROUNDS", defaultRounds),
		Iterations:            getEnvInt("PCA_ITERATIONS", defaultIterations),
		WebhookRetry:          getEnvInt("PCA_WEBHOOK_RETRY", DefaultWebhookRetry),
		WebhookSleep:          getEnvInt("PCA_WEBHOOK_SLEEP", DefaultWebhookSleep),
		WebhookTimeout:        getEnvInt("PCA_WEBHOOK_TIMEOUT", DefaultWebhookTimeout),
		WebhookPingOnPreStart: getEnvBool("PCA_WEBHOOK_PING_ON_PRESTART", DefaultWebhookPingOnPreStart),
		CPUHotplugWatchdog:    getEnvBool("PCA_CPU_HOTPLUG_WATCHDOG", DefaultCPUHotplugWatchdog),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
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
