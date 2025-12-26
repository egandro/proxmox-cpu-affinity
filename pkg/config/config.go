package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	// Logging defaults
	ConstantLogDir      = "/var/log"
	ConstantLogFilename = "proxmox-cpu-affinity.log"
	ConstantLogFile     = ConstantLogDir + "/" + ConstantLogFilename

	// Proxmox defaults
	ConstantQemuServerPidDir   = "/var/run/qemu-server"
	ConstantConfigFilename     = "/etc/default/proxmox-cpu-affinity"
	ConstantSocketFile         = "/var/run/proxmox-cpu-affinity.sock"
	ConstantProxmoxConfigDir   = "/etc/pve"
	ConstantHookScriptFilename = "proxmox-cpu-affinity-hook"

	// Executor commands
	CommandProxmoxQM        = "/usr/sbin/qm"
	CommandProxmoxHaManager = "/usr/sbin/ha-manager"
	CommandSystemPS         = "/usr/bin/ps"
	CommandSystemTaskSet    = "/usr/bin/taskset"
	CommandSystemPGrep      = "/usr/bin/pgrep"

	// ConstantCPUHotplugBatchWindow is the time window to group hotplug events.
	// When a CPU event occurs, we wait this long for subsequent events
	// to arrive (debouncing) before triggering a topology recalculation.
	ConstantCPUHotplugBatchWindow = 5 * time.Second

	ConstantMaxCalculationRankingDuration = 2 * time.Minute

	ConstantSocketTimeout = 5 * time.Second

	// Socket defaults
	DefaultSocketRetry          = 10
	DefaultSocketSleep          = 10 // in seconds
	DefaultSocketTimeout        = 30 // in seconds
	DefaultSocketPingOnPreStart = true

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

	// logger
	DefaultLogLevel = "info"

	DefaultCPUHotplugWatchdog = true
)

// AdaptiveCpuInfoParameters calculates measurement parameters based on CPU count.
// Larger systems use reduced parameters to avoid excessive benchmark time.
// Returns (rounds, iterations).
func AdaptiveCpuInfoParameters() (int, int) {
	cpuCount := numCPU()

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
	LogLevel             string
	LogFile              string
	SocketFile           string
	Rounds               int
	Iterations           int
	SocketRetry          int
	SocketSleep          int // in seconds
	SocketTimeout        int // in seconds
	SocketPingOnPreStart bool
	CPUHotplugWatchdog   bool
}

func Load(filename string) *Config {
	if filename == "" {
		filename = ConstantConfigFilename
	}
	_ = godotenv.Load(filename)

	// Get adaptive defaults based on CPU count, allowing user overrides
	// TODO: reload this after a CPU hotplug event
	defaultRounds, defaultIterations := AdaptiveCpuInfoParameters()

	return &Config{
		LogLevel:             getEnv("PCA_LOG_LEVEL", DefaultLogLevel),
		LogFile:              getEnv("PCA_LOG_FILE", ConstantLogFile),
		SocketFile:           getEnv("PCA_SOCKET_FILE", ConstantSocketFile),
		Rounds:               getEnvInt("PCA_ROUNDS", defaultRounds),
		Iterations:           getEnvInt("PCA_ITERATIONS", defaultIterations),
		SocketRetry:          getEnvInt("PCA_SOCKET_RETRY", DefaultSocketRetry),
		SocketSleep:          getEnvInt("PCA_SOCKET_SLEEP", DefaultSocketSleep),
		SocketTimeout:        getEnvInt("PCA_SOCKET_TIMEOUT", DefaultSocketTimeout),
		SocketPingOnPreStart: getEnvBool("PCA_SOCKET_PING_ON_PRESTART", DefaultSocketPingOnPreStart),
		CPUHotplugWatchdog:   getEnvBool("PCA_CPU_HOTPLUG_WATCHDOG", DefaultCPUHotplugWatchdog),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// numCPU returns the number of physical CPUs found in sysfs.
// We duplicate this logic here to avoid a circular dependency with pkg/cpuinfo.
func numCPU() int {
	matches, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil || len(matches) == 0 {
		return runtime.NumCPU()
	}
	return len(matches)
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
