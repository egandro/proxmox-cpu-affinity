package scheduler

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/proxmox"
)

// CPUSet and schedSetaffinity are defined in affinity_linux.go for Linux
// and affinity_other.go for other platforms.

// affinityProvider defines the internal interface for affinity operations.
type affinityProvider interface {
	ApplyAffinity(vmid int, pid int, config *proxmox.VmConfig) (string, error)
}

type cpuInfoProvider interface {
	GetCoreRanking() ([]cpuinfo.CoreRanking, error)
}

// SystemAffinityOps defines an interface for system-level affinity operations.
type SystemAffinityOps interface {
	SchedSetaffinity(pid int, mask *CPUSet) error
	GetProcessThreads(pid int) ([]int, error)
	GetChildProcesses(pid int) ([]int, error)
}

type defaultSystemAffinityOps struct{}

func (s *defaultSystemAffinityOps) SchedSetaffinity(pid int, mask *CPUSet) error {
	return schedSetaffinity(pid, mask)
}

func (s *defaultSystemAffinityOps) GetProcessThreads(pid int) ([]int, error) {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/task", pid))
	if err != nil {
		return nil, err
	}
	var tids []int
	for _, e := range entries {
		if tid, err := strconv.Atoi(e.Name()); err == nil {
			tids = append(tids, tid)
		}
	}
	return tids, nil
}

func (s *defaultSystemAffinityOps) GetChildProcesses(pid int) ([]int, error) {
	tids, err := s.GetProcessThreads(pid)
	if err != nil {
		return nil, err
	}
	var children []int
	for _, tid := range tids {
		content, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, tid))
		if err != nil {
			continue
		}
		for _, f := range strings.Fields(string(content)) {
			if child, err := strconv.Atoi(f); err == nil {
				children = append(children, child)
			}
		}
	}
	return children, nil
}

type defaultAffinityProvider struct {
	cpuInfo    cpuInfoProvider
	sys        SystemAffinityOps
	affinityMu sync.Mutex
	lastIndex  int
	config     *config.Config
}

func newAffinityProvider(cfg *config.Config, cpuInfo cpuInfoProvider) affinityProvider {
	return &defaultAffinityProvider{
		cpuInfo:   cpuInfo,
		sys:       &defaultSystemAffinityOps{},
		lastIndex: 0,
		config:    cfg,
	}
}

func (a *defaultAffinityProvider) ApplyAffinity(vmid int, pid int, config *proxmox.VmConfig) (string, error) {
	a.affinityMu.Lock()
	defer a.affinityMu.Unlock()

	r, err := a.cpuInfo.GetCoreRanking()
	if err != nil {
		return "", err
	}
	if len(r) == 0 {
		return "", fmt.Errorf("core ranking calculation returned empty results, cannot apply affinity")
	}

	count := config.Cores * config.Sockets
	if count == 0 {
		return "", fmt.Errorf("invalid VM configuration: cores * sockets is 0")
	}
	max := len(r)
	if count >= max {
		slog.Warn("Skipping affinity: requested cores exceed available", "vmid", vmid, "requested", count, "available", max)
		return "", nil
	}

	a.lastIndex++
	if a.lastIndex >= max {
		a.lastIndex = 0
	}

	var res []string
	var mask CPUSet

	primary := r[a.lastIndex]
	res = append(res, strconv.Itoa(primary.CPU))
	mask.Set(primary.CPU)

	slog.Info("Calculating affinity", "vmid", vmid, "primary_index", a.lastIndex, "primary_cpu", primary.CPU, "requested_cores", count)

	for i := 0; i < count-1; i++ {
		if i >= len(primary.Ranking) {
			break
		}
		neighbor := primary.Ranking[i]
		res = append(res, strconv.Itoa(neighbor.CPU))
		mask.Set(neighbor.CPU)
		slog.Info("Selected neighbor", "vmid", vmid, "cpu", neighbor.CPU, "latency_ns", neighbor.LatencyNS)
	}

	slog.Info("Applying affinity", "vmid", vmid, "cpus", res)

	// Collect all PIDs/TIDs to apply affinity to
	pidsToUpdate := a.collectPidsToUpdate(pid)

	allPids := make([]int, 0, len(pidsToUpdate))
	for targetPID := range pidsToUpdate {
		allPids = append(allPids, targetPID)
		if err := a.sys.SchedSetaffinity(targetPID, &mask); err != nil {
			slog.Error("Failed to set process affinity", "vmid", vmid, "pid", targetPID, "error", err)
			// We continue trying other threads even if one fails
		}
	}
	sort.Ints(allPids)

	affinityStr := strings.Join(res, ",")
	slog.Info("Successfully applied affinity", "vmid", vmid, "main_pid", pid, "tids", allPids, "affinity", affinityStr)
	return affinityStr, nil
}

func (a *defaultAffinityProvider) collectPidsToUpdate(pid int) map[int]struct{} {
	pidsToUpdate := make(map[int]struct{})

	// All threads of the main process
	if tids, err := a.sys.GetProcessThreads(pid); err != nil {
		slog.Warn("Failed to get threads, falling back to main PID", "pid", pid, "error", err)
		pidsToUpdate[pid] = struct{}{}
	} else {
		for _, tid := range tids {
			pidsToUpdate[tid] = struct{}{}
		}
	}

	// All child processes and their threads
	if children, err := a.sys.GetChildProcesses(pid); err == nil {
		for _, child := range children {
			pidsToUpdate[child] = struct{}{}
			if childThreads, err := a.sys.GetProcessThreads(child); err == nil {
				for _, ct := range childThreads {
					pidsToUpdate[ct] = struct{}{}
				}
			}
		}
	}
	return pidsToUpdate
}
