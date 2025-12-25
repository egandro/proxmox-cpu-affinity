package cpuinfo

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// lockToCPU is defined in cpuinfo_linux.go for Linux
// and cpuinfo_other.go for other platforms.

// Provider defines the interface for CPU topology and ranking operations.
type Provider interface {
	Update(rounds int, iterations int, onProgress func(int, int)) error
	GetCoreRanking() ([]CoreRanking, error)
	CalculateRanking(rounds, iterations int, timeout time.Duration) error
	DetectTopology() ([]CoreInfo, error)
	SelectCores(requestedCores int) ([]int, error)
}

// topologyDetector is a function that returns the current CPU topology.
type topologyDetector func() ([]CoreInfo, error)

// latencyMeasurer is a function that measures latency between two CPUs.
type latencyMeasurer func(cpuA, cpuB, iter int) (float64, error)

// CPUInfo handles CPU topology detection and latency measurement.
type CPUInfo struct {
	mu        sync.RWMutex
	cache     []CoreRanking
	lastIndex int
	detector  topologyDetector
	measurer  latencyMeasurer
}

// New creates a new CPUInfo instance.
func New() Provider {
	return &CPUInfo{
		detector: detectTopologySystem,
		measurer: measureSingleLink,
	}
}

// CoreInfo represents the CPU topology using standard Linux terminology
// - CPU: The logical processor ID (used by `taskset -c`).
// - Socket: The physical package ID.
// - Core: The physical core ID within the socket.
type CoreInfo struct {
	CPU    int `json:"cpu"`    // Logical Processor
	Socket int `json:"socket"` // Physical Socket
	Core   int `json:"core"`   // Physical Core
}

// Neighbor represents a target core and the cost (latency) to reach it.
type Neighbor struct {
	CPU       int     `json:"cpu"`
	Socket    int     `json:"socket"`
	Core      int     `json:"core"`
	LatencyNS float64 `json:"latency_ns"`
	StdDev    float64 `json:"std_dev"`
}

// CoreRanking contains a source core and its neighbors sorted by affinity (latency).
type CoreRanking struct {
	CPU     int        `json:"cpu"` // Source Logical Processor
	Ranking []Neighbor `json:"ranking"`
}

// CalculateRanking performs the update with a timeout and logs the summary.
func (c *CPUInfo) CalculateRanking(rounds, iterations int, timeout time.Duration) error {
	start := time.Now()
	slog.Info("Calculating core-to-core ranking", "rounds", rounds, "iterations", iterations)

	done := make(chan error, 1)

	onProgress := func(round, total int) {
		slog.Debug("Ranking calculation progress", "round", round, "total", total)
	}

	go func() {
		done <- c.Update(rounds, iterations, onProgress)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("error calculating ranking: %w", err)
		}
	case <-time.After(timeout):
		return fmt.Errorf("calculation timed out after %v (rounds=%d, iterations=%d). This might be a bug/timing issue. Please adjust PCA_ROUNDS/PCA_ITERATIONS", timeout, rounds, iterations)
	}

	rankings, err := c.GetCoreRanking()
	if err != nil {
		return fmt.Errorf("error getting cpuinfo core ranking: %w", err)
	}

	statsJSON, _ := json.Marshal(SummarizeRankings(rankings))
	slog.Info("CPU topology ranking calculated", "duration", time.Since(start).Round(time.Millisecond), "summary", string(statsJSON))
	return nil
}

// Update measures the latency between cores and updates the internal cache.
// rounds: Number of full measurement passes to average.
// iterations: Ping-pongs per measurement.
// onProgress: Optional callback function invoked before each round (round, total).
func (c *CPUInfo) Update(rounds int, iterations int, onProgress func(int, int)) error {
	// 1. Discover Topology
	topology, err := c.detector()
	if err != nil {
		return fmt.Errorf("error detecting topology: %w", err)
	}

	numCores := len(topology)
	latSums := make([]float64, numCores*numCores)
	latSqSums := make([]float64, numCores*numCores)

	// 2. Measure Accumulator (Linearized Matrix)
	for r := 0; r < rounds; r++ {
		if onProgress != nil {
			onProgress(r+1, rounds)
		}
		for i, src := range topology {
			for j, dst := range topology {
				if i == j {
					continue
				}
				// Measure latency between logical CPU i and logical CPU j
				lat, err := c.measurer(src.CPU, dst.CPU, iterations)
				if err != nil {
					return fmt.Errorf("failed to measure latency between CPU %d and %d: %w", src.CPU, dst.CPU, err)
				}
				latSums[i*numCores+j] += lat
				latSqSums[i*numCores+j] += lat * lat
			}
		}
	}

	// 3. Aggregate and Sort Results
	var finalResults []CoreRanking

	for i, src := range topology {
		var neighbors []Neighbor

		for j, dst := range topology {
			if i == j {
				continue
			}

			avgLat := latSums[i*numCores+j] / float64(rounds)

			variance := (latSqSums[i*numCores+j] / float64(rounds)) - (avgLat * avgLat)
			if variance < 0 {
				variance = 0
			}

			neighbors = append(neighbors, Neighbor{
				CPU:       dst.CPU,
				Socket:    dst.Socket,
				Core:      dst.Core,
				LatencyNS: avgLat,
				StdDev:    math.Sqrt(variance),
			})
		}

		// Sort: Low Latency (Happy) -> High Latency (Unhappy)
		sort.Slice(neighbors, func(a, b int) bool {
			return neighbors[a].LatencyNS < neighbors[b].LatencyNS
		})

		finalResults = append(finalResults, CoreRanking{
			CPU:     src.CPU,
			Ranking: neighbors,
		})
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = finalResults
	// Ensure lastIndex is within bounds if topology shrank
	if len(c.cache) > 0 {
		c.lastIndex = c.lastIndex % len(c.cache)
	} else {
		c.lastIndex = 0
	}
	return nil
}

// GetCoreRanking returns the cached core ranking.
func (c *CPUInfo) GetCoreRanking() ([]CoreRanking, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.cache) == 0 {
		return nil, fmt.Errorf("cache is empty, you have to call Update() first")
	}
	return c.cache, nil
}

// SelectCores returns a list of CPU IDs for the next VM, rotating through available cores.
// This method is thread-safe to handle concurrent access, specifically when CPU hotplug
// events trigger a topology update (changing the cache) while affinity is being requested.
func (c *CPUInfo) SelectCores(requestedCores int) ([]int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) == 0 {
		return nil, fmt.Errorf("core ranking cache is empty")
	}

	if requestedCores <= 0 {
		return nil, fmt.Errorf("requested cores must be greater than 0")
	}

	max := len(c.cache)
	if requestedCores > max {
		return nil, fmt.Errorf("requested cores %d exceed available %d", requestedCores, max)
	}

	c.lastIndex = (c.lastIndex + 1) % max

	primary := c.cache[c.lastIndex]
	res := make([]int, 0, requestedCores)
	res = append(res, primary.CPU)

	for i := 0; i < requestedCores-1 && i < len(primary.Ranking); i++ {
		res = append(res, primary.Ranking[i].CPU)
	}

	return res, nil
}

// DetectTopology reads Linux sysfs to find CPU topology.
func (c *CPUInfo) DetectTopology() ([]CoreInfo, error) {
	return c.detector()
}

// NumCPU returns the number of physical CPUs found in sysfs.
// It is a replacement for runtime.NumCPU() which only returns logical CPUs available to the process.
// we need this to mitigate for a CPU hotplug event
func NumCPU() int {
	matches, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil || len(matches) == 0 {
		return runtime.NumCPU()
	}
	return len(matches)
}

func detectTopologySystem() ([]CoreInfo, error) {
	var cores []CoreInfo

	// We look at the sysfs directly to find all present CPUs.
	matches, err := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	if err != nil {
		return nil, err
	}

	for _, path := range matches {
		// Extract CPU ID from path (e.g. /sys/devices/system/cpu/cpu0 -> 0)
		cpuIDStr := strings.TrimPrefix(filepath.Base(path), "cpu")
		i, err := strconv.Atoi(cpuIDStr)
		if err != nil {
			continue
		}

		// 1. Socket ID (physical_package_id)
		socketID, err := readSysFSInt(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/physical_package_id", i))
		if err != nil {
			// Skip offline/inaccessible CPUs
			continue
		}

		// 2. Physical Core ID (core_id)
		coreID, err := readSysFSInt(fmt.Sprintf("/sys/devices/system/cpu/cpu%d/topology/core_id", i))
		if err != nil {
			coreID = -1
		}

		cores = append(cores, CoreInfo{
			CPU:    i, // This matches `taskset -c` ID
			Socket: socketID,
			Core:   coreID,
		})
	}

	// Ensure deterministic order
	sort.Slice(cores, func(i, j int) bool {
		return cores[i].CPU < cores[j].CPU
	})

	return cores, nil
}

func measureSingleLink(cpuA, cpuB, iter int) (float64, error) {
	var wg sync.WaitGroup
	wg.Add(2)

	var barrier sync.WaitGroup
	barrier.Add(2)

	var signal int32 = 0
	var errMutex sync.Mutex
	var firstErr error

	go func() {
		defer wg.Done()
		if err := lockToCPU(cpuA); err != nil {
			errMutex.Lock()
			if firstErr == nil {
				firstErr = err
			}
			errMutex.Unlock()
			barrier.Done()
			return
		}
		barrier.Done()
		barrier.Wait()
		if firstErr != nil {
			return
		}
		for k := 0; k < iter; k++ {
			for atomic.LoadInt32(&signal) != 0 {
			}
			atomic.StoreInt32(&signal, 1)
		}
	}()

	go func() {
		defer wg.Done()
		if err := lockToCPU(cpuB); err != nil {
			errMutex.Lock()
			if firstErr == nil {
				firstErr = err
			}
			errMutex.Unlock()
			barrier.Done()
			return
		}
		barrier.Done()
		barrier.Wait()
		if firstErr != nil {
			return
		}
		for k := 0; k < iter; k++ {
			for atomic.LoadInt32(&signal) != 1 {
			}
			atomic.StoreInt32(&signal, 0)
		}
	}()

	start := time.Now()
	wg.Wait()
	duration := time.Since(start)

	if firstErr != nil {
		return 0, firstErr
	}

	return float64(duration.Nanoseconds()) / float64(iter*2), nil
}

func readSysFSInt(path string) (int, error) {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		realPath = path
	}
	data, err := os.ReadFile(realPath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// LatencyStat holds latency metrics for a specific link.
type LatencyStat struct {
	LatencyNS float64 `json:"latency_ns"`
	StdDev    float64 `json:"std_dev"`
	SrcCPU    int     `json:"src_cpu"`
	DstCPU    int     `json:"dst_cpu"`
}

// TopologyStats contains statistics about the CPU topology and latency.
type TopologyStats struct {
	CPUCount      int         `json:"cpu_count"`
	SocketCount   int         `json:"socket_count"`
	Min           LatencyStat `json:"min"` // Best performance (lowest latency)
	Max           LatencyStat `json:"max"` // Worst performance (highest latency)
	MeanLatencyNS float64     `json:"mean_latency_ns"`
}

// SummarizeRankings returns statistics about the core rankings.
func SummarizeRankings(rankings []CoreRanking) TopologyStats {
	if len(rankings) == 0 {
		return TopologyStats{}
	}

	var stats TopologyStats
	stats.Min.LatencyNS = 1e18
	stats.Max.LatencyNS = -1.0

	var totalLat float64
	var count int

	sockets := make(map[int]struct{})
	cpus := make(map[int]struct{})

	for _, r := range rankings {
		cpus[r.CPU] = struct{}{}
		for _, n := range r.Ranking {
			sockets[n.Socket] = struct{}{}
			cpus[n.CPU] = struct{}{}

			val := n.LatencyNS
			totalLat += val
			count++

			if val < stats.Min.LatencyNS {
				stats.Min.LatencyNS = val
				stats.Min.StdDev = n.StdDev
				stats.Min.SrcCPU = r.CPU
				stats.Min.DstCPU = n.CPU
			}
			if val > stats.Max.LatencyNS {
				stats.Max.LatencyNS = val
				stats.Max.StdDev = n.StdDev
				stats.Max.SrcCPU = r.CPU
				stats.Max.DstCPU = n.CPU
			}
		}
	}

	stats.CPUCount = len(cpus)
	stats.SocketCount = len(sockets)

	if count > 0 {
		stats.MeanLatencyNS = totalLat / float64(count)
	} else {
		stats.Min.LatencyNS = 0
		stats.Max.LatencyNS = 0
	}

	// round := func(v float64) float64 {
	// 	return math.Round(v*100) / 100
	// }

	// stats.Min.LatencyNS = round(stats.Min.LatencyNS)
	// stats.Min.StdDev = round(stats.Min.StdDev)
	// stats.Max.LatencyNS = round(stats.Max.LatencyNS)
	// stats.Max.StdDev = round(stats.Max.StdDev)
	// stats.MeanLatencyNS = round(stats.MeanLatencyNS)

	return stats
}
