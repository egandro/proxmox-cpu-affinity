package scheduler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/proxmox"
)

// Scheduler defines the interface for VM scheduling operations.
type Scheduler interface {
	VmStarted(ctx context.Context, vmid int) (interface{}, error)
}

// ProxmoxClient defines the interface for Proxmox operations.
type ProxmoxClient interface {
	GetVmConfig(ctx context.Context, vmid int) (*proxmox.VmConfig, error)
	GetVmPid(ctx context.Context, vmid int) (int, error)
}

// scheduler implements the Scheduler interface.
type scheduler struct {
	proxmox  ProxmoxClient
	affinity affinityProvider
}

// New creates a new scheduler.
func New(cfg *config.Config, cpuInfo cpuinfo.Provider) (Scheduler, error) {
	p, err := proxmox.New()
	if err != nil {
		return nil, err
	}
	return &scheduler{
		proxmox:  p,
		affinity: newAffinityProvider(cfg, cpuInfo),
	}, nil
}

// VmStarted handles the logic for starting a VM with affinity.
func (s *scheduler) VmStarted(ctx context.Context, vmid int) (interface{}, error) {
	slog.Info("VmStarted called", "vmid", vmid)

	config, err := s.proxmox.GetVmConfig(ctx, vmid)
	if err != nil {
		slog.Error("Error getting VM config", "vmid", vmid, "error", err)
		return nil, err
	}

	pid, err := s.proxmox.GetVmPid(ctx, vmid)
	if err != nil {
		slog.Error("Error checking if VM is running", "vmid", vmid, "error", err)
		return nil, err
	}
	if pid == -1 {
		return nil, fmt.Errorf("VM %d is not running", vmid)
	}

	if config.HookScript == "" {
		slog.Warn("VM has no hookscript configured", "vmid", vmid)
	}

	if config.Affinity != "" {
		slog.Info("VM has existing affinity configuration", "vmid", vmid, "affinity", config.Affinity)
		return map[string]interface{}{"action": fmt.Sprintf("vm has an affinity configuration %s", config.Affinity)}, nil
	}

	affinity, err := s.affinity.ApplyAffinity(ctx, vmid, pid, config)
	if err != nil {
		slog.Error("Error setting affinity", "vmid", vmid, "error", err)
		return nil, err
	}

	return map[string]interface{}{"action": fmt.Sprintf("new affinity: %s", affinity)}, nil
}
