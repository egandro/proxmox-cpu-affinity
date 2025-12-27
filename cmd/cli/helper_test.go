package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHookPath(t *testing.T) {
	path := getHookPath("local")
	expected := fmt.Sprintf("local:snippets/%s", config.ConstantHookScriptFilename)
	assert.Equal(t, expected, path)
}

func TestIsValidStorage(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"local", true},
		{"local-lvm", true},
		{"my_storage", true},
		{"", false},
		{".", false},
		{"..", false},
		{"path/to/storage", false},
		{"storage/with/slash", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidStorage(tt.input))
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"123", true},
		{"0", true},
		{"-1", false},
		{"abc", false},
		{"12.34", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.valid, isNumeric(tt.input))
		})
	}
}

func TestEnsureProxmoxHost(t *testing.T) {
	// Check if the directory actually exists on the test runner
	_, err := os.Stat(config.ConstantProxmoxConfigDir)
	exists := !os.IsNotExist(err)

	err = ensureProxmoxHost()
	if exists {
		assert.NoError(t, err)
	} else {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be run on a Proxmox VE host")
	}
}

func TestResolveSocketPath(t *testing.T) {
	// Test with flag provided
	assert.Equal(t, "/tmp/flag.sock", resolveSocketPath("/tmp/flag.sock"))

	// Test with env var (simulating config load)
	key := "PCA_SOCKET_FILE"
	original, exists := os.LookupEnv(key)
	defer func() {
		if exists {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	}()

	expectedEnvPath := "/tmp/env.sock"
	_ = os.Setenv(key, expectedEnvPath)

	assert.Equal(t, expectedEnvPath, resolveSocketPath(""))
}

func TestSendSocketRequest(t *testing.T) {
	// Create a temp socket
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	// Start a dummy server
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		var req SocketRequest
		if err := json.NewDecoder(conn).Decode(&req); err != nil {
			return
		}

		resp := SocketResponse{
			Status: "ok",
			Data:   "pong",
		}
		_ = json.NewEncoder(conn).Encode(resp)
	}()

	// Test successful request
	resp, err := sendSocketRequest(socketPath, SocketRequest{Command: "ping"})
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "pong", resp.Data)

	// Test connection failure
	_, err = sendSocketRequest(filepath.Join(tmpDir, "nonexistent.sock"), SocketRequest{Command: "ping"})
	assert.Error(t, err)
}

func TestGetCPUModelName(t *testing.T) {
	tmpDir := t.TempDir()
	cpuInfoPath := filepath.Join(tmpDir, "cpuinfo")

	content := `processor	: 0
vendor_id	: GenuineIntel
cpu family	: 6
model		: 85
model name	: Intel(R) Xeon(R) Gold 6130 CPU @ 2.10GHz
stepping	: 4
microcode	: 0x2000069
`
	err := os.WriteFile(cpuInfoPath, []byte(content), 0600)
	require.NoError(t, err)

	assert.Equal(t, "Intel(R) Xeon(R) Gold 6130 CPU @ 2.10GHz", getCPUModelName(cpuInfoPath))
	assert.Equal(t, "Unknown CPU", getCPUModelName(filepath.Join(tmpDir, "missing")))
}
