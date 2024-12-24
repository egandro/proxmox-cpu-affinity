package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
)

// Executor defines an interface for executing system commands.
// This allows mocking the actual execution in tests.
type Executor interface {
	Output(ctx context.Context, name string, arg ...string) ([]byte, error)
}

// executor is the standard implementation using os/exec.
type executor struct{}

func (d *executor) Output(ctx context.Context, name string, arg ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, arg...)
	return cmd.Output()
}

// SystemOps defines an interface for file system and process operations.
type SystemOps interface {
	ReadFile(name string) ([]byte, error)
	ProcessExists(pid int) bool
}

// systemOps implements SystemOps using the os and syscall packages.
type systemOps struct{}

func (s *systemOps) ReadFile(name string) ([]byte, error) {
	// #nosec G304 -- The path is constructed from a trusted directory and integer VMID in the caller.
	return os.ReadFile(filepath.Clean(name))
}

func (s *systemOps) ProcessExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// proxmox handles interactions with the Proxmox API via pvesh.
type proxmox struct {
	nodeName string
	executor Executor
	sys      SystemOps
}

// New creates a new proxmox instance.
// It resolves the hostname once during initialization.
func New() (*proxmox, error) {
	// Determine the current hostname (node name)
	fullHostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}
	// Proxmox node names are typically the short hostname (before the first dot)
	nodeName := strings.Split(fullHostname, ".")[0]

	return &proxmox{
		nodeName: nodeName,
		executor: &executor{},
		sys:      &systemOps{},
	}, nil
}

// VmConfig represents the configuration of a Proxmox QEMU VM.
type VmConfig struct {
	Cores      int    `json:"cores"`
	Sockets    int    `json:"sockets"`
	Affinity   string `json:"affinity,omitempty"`
	HookScript string `json:"hookscript,omitempty"`
}

// GetVmConfig retrieves the configuration of a Proxmox VM by its ID.
// It executes: /usr/bin/pvesh get /nodes/HOSTNAME/qemu/VMID/config --output-format json-pretty
func (p *proxmox) GetVmConfig(ctx context.Context, vmid int) (*VmConfig, error) {
	// Construct the pvesh command
	argPath := fmt.Sprintf("/nodes/%s/qemu/%d/config", p.nodeName, vmid)

	output, err := p.executor.Output(ctx, "/usr/bin/pvesh", "get", argPath, "--output-format", "json-pretty")
	if err != nil {
		return nil, fmt.Errorf("pvesh command failed: %w", err)
	}

	slog.Debug("VM config raw output", "vmid", vmid, "output", string(output))

	var config VmConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return nil, fmt.Errorf("failed to parse VM config JSON: %w", err)
	}

	if config.Cores == 0 {
		config.Cores = 1
	}
	if config.Sockets == 0 {
		config.Sockets = 1
	}

	configJSON, _ := json.Marshal(config)
	slog.Debug("Parsed VM config", "config", string(configJSON))

	return &config, nil
}

// GetVmPid checks if the VM is running and returns its PID. Returns -1 if not running.
func (p *proxmox) GetVmPid(_ context.Context, vmid int) (int, error) {
	pidPath := fmt.Sprintf("%s/%d.pid", config.DefaultQemuServerPidDir, vmid)

	content, err := p.sys.ReadFile(pidPath)
	if os.IsNotExist(err) {
		return -1, nil
	}
	if err != nil {
		return -1, fmt.Errorf("failed to read pid file: %w", err)
	}

	pidStr := strings.TrimSpace(string(content))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return -1, fmt.Errorf("failed to parse pid from file: %w", err)
	}

	if !p.sys.ProcessExists(pid) {
		return -1, nil
	}
	return pid, nil
}
