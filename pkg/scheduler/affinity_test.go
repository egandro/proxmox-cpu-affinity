package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/proxmox"
)

// MockCpuInfoProvider mocks the cpuInfoProvider interface.
type MockCpuInfoProvider struct {
	mock.Mock
}

func (m *MockCpuInfoProvider) Update(rounds int, iterations int) error {
	args := m.Called(rounds, iterations)
	return args.Error(0)
}

func (m *MockCpuInfoProvider) GetCoreRanking() ([]cpuinfo.CoreRanking, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpuinfo.CoreRanking), args.Error(1)
}

func (m *MockCpuInfoProvider) SelectCPUs(vmid int, requestedCPUs int) ([]int, error) {
	args := m.Called(vmid, requestedCPUs)
	return args.Get(0).([]int), args.Error(1)
}

// MockSystemAffinityOps mocks the SystemAffinityOps interface.
type MockSystemAffinityOps struct {
	mock.Mock
}

func (m *MockSystemAffinityOps) SchedSetaffinity(pid int, mask *CPUSet) error {
	args := m.Called(pid, mask)
	return args.Error(0)
}

func (m *MockSystemAffinityOps) GetProcessThreads(pid int) ([]int, error) {
	args := m.Called(pid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockSystemAffinityOps) GetChildProcesses(pid int) ([]int, error) {
	args := m.Called(pid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]int), args.Error(1)
}

func TestApplyAffinity(t *testing.T) {
	tests := []struct {
		name           string
		vmid           int
		pid            int
		config         *proxmox.VmConfig
		expectedRes    string
		expectError    bool
		expectedErrMsg string
		setupMockSys   func(*MockSystemAffinityOps)
		setupMockCpu   func(*MockCpuInfoProvider)
	}{
		{
			name: "Success - 2 Cores",
			vmid: 100,
			pid:  12345,
			config: &proxmox.VmConfig{
				Cores:   2,
				Sockets: 1,
			},
			expectedRes: "1,0",
			setupMockCpu: func(m *MockCpuInfoProvider) {
				m.On("SelectCPUs", 100, 2).Return([]int{1, 0}, nil)
			},
			setupMockSys: func(m *MockSystemAffinityOps) {
				m.On("GetProcessThreads", 12345).Return([]int{12345}, nil)
				m.On("GetChildProcesses", 12345).Return([]int{}, nil)
				m.On("SchedSetaffinity", 12345, mock.MatchedBy(func(mask *CPUSet) bool {
					return mask.IsSet(1) && mask.IsSet(0) && !mask.IsSet(2)
				})).Return(nil)
			},
		},
		{
			name: "Affinity Error (invalid config)",
			vmid: 102,
			pid:  12347,
			config: &proxmox.VmConfig{
				Cores:   0,
				Sockets: 0,
			},
			expectError:    true,
			expectedErrMsg: "invalid VM configuration",
			expectedRes:    "",
			setupMockCpu: func(m *MockCpuInfoProvider) {
				// Should not be called
			},
			setupMockSys: func(m *MockSystemAffinityOps) {
				// Should not be called
			},
		},
		{
			name: "Success - SchedSetaffinity Failed (Logged Only)",
			vmid: 105,
			pid:  12350,
			config: &proxmox.VmConfig{
				Cores:   1,
				Sockets: 1,
			},
			expectError: false,
			expectedRes: "1",
			setupMockCpu: func(m *MockCpuInfoProvider) {
				m.On("SelectCPUs", 105, 1).Return([]int{1}, nil)
			},
			setupMockSys: func(m *MockSystemAffinityOps) {
				m.On("GetProcessThreads", 12350).Return([]int{12350}, nil)
				m.On("GetChildProcesses", 12350).Return([]int{}, nil)
				m.On("SchedSetaffinity", 12350, mock.Anything).Return(errors.New("sys error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCpu := new(MockCpuInfoProvider)
			mockSys := new(MockSystemAffinityOps)

			p := &defaultAffinityProvider{
				cpuInfo: mockCpu,
				sys:     mockSys,
				config:  &config.Config{Rounds: 1, Iterations: 1},
			}

			// Setup Ranking Mock
			if tt.setupMockCpu != nil {
				tt.setupMockCpu(mockCpu)
			}

			// Setup Sys Mock
			if tt.setupMockSys != nil {
				tt.setupMockSys(mockSys)
			}

			res, err := p.ApplyAffinity(context.Background(), tt.vmid, tt.pid, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRes, res)
			}

			mockCpu.AssertExpectations(t)
			mockSys.AssertExpectations(t)
		})
	}
}
