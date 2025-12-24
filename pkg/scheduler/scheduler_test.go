package scheduler

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/proxmox"
)

// MockProxmoxClient mocks the ProxmoxClient interface.
type MockProxmoxClient struct {
	mock.Mock
}

func (m *MockProxmoxClient) GetVmConfig(vmid int) (*proxmox.VmConfig, error) {
	args := m.Called(vmid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*proxmox.VmConfig), args.Error(1)
}

func (m *MockProxmoxClient) GetVmPid(vmid int) (int, error) {
	args := m.Called(vmid)
	return args.Int(0), args.Error(1)
}

// MockAffinityProvider mocks the affinityProvider interface.
type MockAffinityProvider struct {
	mock.Mock
}

func (m *MockAffinityProvider) GetCoreRanking() ([]cpuinfo.CoreRanking, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpuinfo.CoreRanking), args.Error(1)
}

func (m *MockAffinityProvider) ApplyAffinity(vmid int, pid int, config *proxmox.VmConfig) (string, error) {
	args := m.Called(vmid, pid, config)
	return args.String(0), args.Error(1)
}

func TestVmStarted(t *testing.T) {
	tests := []struct {
		name           string
		vmid           int
		config         *proxmox.VmConfig
		configErr      error
		affinityResult string
		affinityErr    error
		pid            int
		runningErr     error
		expectError    bool
		expectAction   string
	}{
		{
			name: "Success - Apply Affinity",
			vmid: 100,
			config: &proxmox.VmConfig{
				Cores:      4,
				HookScript: "local:snippets/hook.pl",
			},
			affinityResult: "0-3",
			expectAction:   "new affinity: 0-3",
			pid:            12345,
		},
		{
			name: "Success - Hardcoded Affinity",
			vmid: 101,
			config: &proxmox.VmConfig{
				Cores:      4,
				HookScript: "local:snippets/hook.pl",
				Affinity:   "0,1",
			},
			expectAction: "vm has an affinity configuration 0,1",
			pid:          12345,
		},
		{
			name: "Success - Missing HookScript Warning",
			vmid: 104,
			config: &proxmox.VmConfig{
				Cores: 4,
			},
			affinityResult: "0-3",
			expectAction:   "new affinity: 0-3",
			pid:            12345,
		},
		{
			name:        "Error - GetVmConfig Failed",
			vmid:        102,
			configErr:   errors.New("proxmox error"),
			expectError: true,
		},
		{
			name: "Error - VM Not Running",
			vmid: 105,
			config: &proxmox.VmConfig{
				Cores: 4,
			},
			pid:         -1,
			expectError: true,
		},
		{
			name: "Error - ApplyAffinity Failed",
			vmid: 103,
			config: &proxmox.VmConfig{
				Cores:      4,
				HookScript: "local:snippets/hook.pl",
			},
			affinityErr: errors.New("affinity error"),
			expectError: true,
			pid:         12345,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProxmox := new(MockProxmoxClient)
			mockAffinity := new(MockAffinityProvider)

			// Setup expectations
			mockProxmox.On("GetVmConfig", tt.vmid).Return(tt.config, tt.configErr)

			if tt.configErr == nil {
				// GetCoreRanking is called before GetVmPid to minimize TOCTOU race window
				mockAffinity.On("GetCoreRanking").Return([]cpuinfo.CoreRanking{{CPU: 0}}, nil)
				mockProxmox.On("GetVmPid", tt.vmid).Return(tt.pid, tt.runningErr)
			}

			if tt.configErr == nil && tt.pid != -1 && (tt.config == nil || tt.config.Affinity == "") {
				mockAffinity.On("ApplyAffinity", tt.vmid, tt.pid, tt.config).Return(tt.affinityResult, tt.affinityErr)
			}

			s := &scheduler{
				proxmox:  mockProxmox,
				affinity: mockAffinity,
			}

			result, err := s.VmStarted(tt.vmid)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				resMap, ok := result.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tt.expectAction, resMap["action"])
			}

			mockProxmox.AssertExpectations(t)
			mockAffinity.AssertExpectations(t)
		})
	}
}
