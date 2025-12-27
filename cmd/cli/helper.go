package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
)

func getHookPath(storage string) string {
	return fmt.Sprintf("%s:snippets/%s", storage, config.ConstantHookScriptFilename)
}

func isValidStorage(s string) bool {
	// Fails if s is empty, is ".", "..", or contains a separator
	if s == "" || s == "." || s == ".." {
		return false
	}
	// If Base(s) changes the string, it meant there were separators
	return filepath.Base(s) == s
}

func isNumeric(s string) bool {
	_, err := strconv.ParseUint(s, 10, 64)
	return err == nil
}

func ensureProxmoxHost() error {
	if _, err := os.Stat(config.ConstantProxmoxConfigDir); os.IsNotExist(err) {
		return fmt.Errorf("this tool must be run on a Proxmox VE host (%s not found)", config.ConstantProxmoxConfigDir)
	}
	return nil
}

// SocketRequest represents the JSON request structure for the service.
type SocketRequest struct {
	Command string `json:"command"`
	VMID    int    `json:"vmid,omitempty"`
}

// SocketResponse represents the JSON response structure from the service.
type SocketResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func resolveSocketPath(flagSocket string) string {
	if flagSocket != "" {
		return flagSocket
	}
	cfg := config.Load(config.ConstantConfigFilename)
	return cfg.SocketFile
}

func sendSocketRequest(socketPath string, req SocketRequest) (*SocketResponse, error) {
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("service is not reachable: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	var resp SocketResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &resp, nil
}

func getCPUModelName(path string) string {
	if path == "" {
		path = config.ConstantProcCpuInfo
	}
	// #nosec G304 -- path is either hardcoded /proc/cpuinfo (ConstantProcCpuInfo) or a test file
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "Unknown CPU"
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, "model name") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown CPU"
}
