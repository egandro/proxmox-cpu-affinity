package proxmox

import (
	"embed"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata
var testData embed.FS

// MockExecutor implements Executor for testing purposes.
type MockExecutor struct {
	OutputFunc func(name string, arg ...string) ([]byte, error)
}

func (m *MockExecutor) Output(name string, arg ...string) ([]byte, error) {
	if m.OutputFunc != nil {
		return m.OutputFunc(name, arg...)
	}
	return nil, fmt.Errorf("MockExecutor.OutputFunc not implemented")
}

// MockSystemOps implements SystemOps for testing purposes.
type MockSystemOps struct {
	ReadFileFunc      func(name string) ([]byte, error)
	ProcessExistsFunc func(pid int) bool
}

func (m *MockSystemOps) ReadFile(name string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(name)
	}
	return nil, fmt.Errorf("MockSystemOps.ReadFileFunc not implemented")
}

func (m *MockSystemOps) ProcessExists(pid int) bool {
	if m.ProcessExistsFunc != nil {
		return m.ProcessExistsFunc(pid)
	}
	return false
}

func TestGetVmConfig(t *testing.T) {
	defaultVmData, err := testData.ReadFile("testdata/default-vm.json")
	assert.NoError(t, err, "failed to read test data")

	affinityVmData, err := testData.ReadFile("testdata/with-affinity.json")
	assert.NoError(t, err, "failed to read affinity test data")

	hookScriptVmData, err := testData.ReadFile("testdata/with-hookscript.json")
	assert.NoError(t, err, "failed to read hookscript test data")

	tests := []struct {
		name        string
		vmid        int
		mockOutput  []byte
		mockError   error
		expectError bool
		check       func(*testing.T, *VmConfig)
	}{
		{
			name:       "Success default-vm",
			vmid:       100,
			mockOutput: defaultVmData,
			check: func(t *testing.T, c *VmConfig) {
				assert.Equal(t, 4, c.Cores)
				assert.Equal(t, 1, c.Sockets)
				assert.Empty(t, c.Affinity)
			},
		},
		{
			name:       "Success with-affinity",
			vmid:       101,
			mockOutput: affinityVmData,
			check: func(t *testing.T, c *VmConfig) {
				assert.Equal(t, 4, c.Cores)
				assert.Equal(t, 1, c.Sockets)
				assert.Equal(t, "0,1,2,3", c.Affinity)
			},
		},
		{
			name:       "Success with-hookscript",
			vmid:       102,
			mockOutput: hookScriptVmData,
			check: func(t *testing.T, c *VmConfig) {
				assert.Equal(t, 4, c.Cores)
				assert.Equal(t, 1, c.Sockets)
				assert.Equal(t, "local:snippets/hookscript.pl", c.HookScript)
			},
		},
		{
			name:        "Error - pvesh command fails",
			vmid:        999,
			mockError:   fmt.Errorf("command failed: exit status 2"),
			expectError: true,
		},
		{
			name:        "Error - invalid JSON response",
			vmid:        998,
			mockOutput:  []byte("not valid json {{{"),
			expectError: true,
		},
		{
			name:       "Success - zero cores defaults to 1",
			vmid:       997,
			mockOutput: []byte(`{"sockets": 2}`),
			check: func(t *testing.T, c *VmConfig) {
				assert.Equal(t, 1, c.Cores)
				assert.Equal(t, 2, c.Sockets)
			},
		},
		{
			name:       "Success - zero sockets defaults to 1",
			vmid:       996,
			mockOutput: []byte(`{"cores": 8}`),
			check: func(t *testing.T, c *VmConfig) {
				assert.Equal(t, 8, c.Cores)
				assert.Equal(t, 1, c.Sockets)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockExecutor{
				OutputFunc: func(name string, arg ...string) ([]byte, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockOutput, nil
				},
			}

			p := &proxmox{executor: mock}

			config, err := p.GetVmConfig(tt.vmid)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if !tt.expectError && tt.check != nil {
				tt.check(t, config)
			}
		})
	}
}

func TestGetVmPid(t *testing.T) {
	tests := []struct {
		name          string
		vmid          int
		mockReadFile  func(string) ([]byte, error)
		mockProcExist func(int) bool
		want          int
		expectError   bool
	}{
		{
			name: "Running",
			vmid: 100,
			mockReadFile: func(name string) ([]byte, error) {
				return []byte("12345"), nil
			},
			mockProcExist: func(pid int) bool {
				return pid == 12345
			},
			want: 12345,
		},
		{
			name: "Not Running (No PID file)",
			vmid: 101,
			mockReadFile: func(name string) ([]byte, error) {
				return nil, os.ErrNotExist
			},
			want: -1,
		},
		{
			name: "Not Running (Process dead)",
			vmid: 102,
			mockReadFile: func(name string) ([]byte, error) {
				return []byte("12345"), nil
			},
			mockProcExist: func(pid int) bool {
				return false
			},
			want: -1,
		},
		{
			name: "Error - ReadFile permission denied",
			vmid: 103,
			mockReadFile: func(name string) ([]byte, error) {
				return nil, os.ErrPermission
			},
			want:        -1,
			expectError: true,
		},
		{
			name: "Error - Invalid PID in file",
			vmid: 104,
			mockReadFile: func(name string) ([]byte, error) {
				return []byte("not-a-number"), nil
			},
			want:        -1,
			expectError: true,
		},
		{
			name: "Running with whitespace in PID file",
			vmid: 105,
			mockReadFile: func(name string) ([]byte, error) {
				return []byte("  54321\n"), nil
			},
			mockProcExist: func(pid int) bool {
				return pid == 54321
			},
			want: 54321,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSys := &MockSystemOps{
				ReadFileFunc:      tt.mockReadFile,
				ProcessExistsFunc: tt.mockProcExist,
			}
			p := &proxmox{sys: mockSys}
			got, err := p.GetVmPid(tt.vmid)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
