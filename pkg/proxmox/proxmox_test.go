package proxmox

import (
	"context"
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
	OutputFunc func(ctx context.Context, name string, arg ...string) ([]byte, error)
}

func (m *MockExecutor) Output(ctx context.Context, name string, arg ...string) ([]byte, error) {
	if m.OutputFunc != nil {
		return m.OutputFunc(ctx, name, arg...)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockExecutor{
				OutputFunc: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockOutput, nil
				},
			}

			p := &proxmox{executor: mock}

			config, err := p.GetVmConfig(context.Background(), tt.vmid)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSys := &MockSystemOps{
				ReadFileFunc:      tt.mockReadFile,
				ProcessExistsFunc: tt.mockProcExist,
			}
			p := &proxmox{sys: mockSys}
			got, err := p.GetVmPid(context.Background(), tt.vmid)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
