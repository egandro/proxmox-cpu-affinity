package scheduler

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sys/unix"

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

// MockSystemAffinityOps mocks the SystemAffinityOps interface.
type MockSystemAffinityOps struct {
	mock.Mock
}

func (m *MockSystemAffinityOps) SchedSetaffinity(pid int, mask *unix.CPUSet) error {
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
	// Define sample rankings: 3 CPUs (0, 1, 2)
	rankings := []cpuinfo.CoreRanking{
		{
			CPU: 0,
			Ranking: []cpuinfo.Neighbor{
				{CPU: 1, LatencyNS: 10},
				{CPU: 2, LatencyNS: 20},
			},
		},
		{
			CPU: 1,
			Ranking: []cpuinfo.Neighbor{
				{CPU: 0, LatencyNS: 10},
				{CPU: 2, LatencyNS: 20},
			},
		},
		{
			CPU: 2,
			Ranking: []cpuinfo.Neighbor{
				{CPU: 0, LatencyNS: 20},
				{CPU: 1, LatencyNS: 20},
			},
		},
	}

	tests := []struct {
		name           string
		vmid           int
		pid            int
		config         *proxmox.VmConfig
		mockRankings   []cpuinfo.CoreRanking
		mockRankingErr error
		mockSysErr     error
		expectedRes    string
		expectError    bool
		setupMockSys   func(*MockSystemAffinityOps)
	}{
		{
			name: "Success - 2 Cores",
			vmid: 100,
			pid:  12345,
			config: &proxmox.VmConfig{
				Cores:   2,
				Sockets: 1,
			},
			mockRankings: rankings,
			// lastIndex starts at 0, increments to 1. CPU 1 + neighbor 0.
			expectedRes: "1,0",
			setupMockSys: func(m *MockSystemAffinityOps) {
				m.On("GetProcessThreads", 12345).Return([]int{12345}, nil)
				m.On("GetChildProcesses", 12345).Return([]int{}, nil)
				m.On("SchedSetaffinity", 12345, mock.MatchedBy(func(mask *unix.CPUSet) bool {
					return mask.IsSet(1) && mask.IsSet(0) && !mask.IsSet(2)
				})).Return(nil)
			},
		},
		{
			name: "Error - Ranking Failed",
			vmid: 102,
			pid:  12347,
			config: &proxmox.VmConfig{
				Cores: 2,
			},
			mockRankingErr: errors.New("ranking failed"),
			expectError:    true,
		},
		{
			name: "Error - Empty Ranking",
			vmid: 103,
			pid:  12348,
			config: &proxmox.VmConfig{
				Cores: 2,
			},
			mockRankings: []cpuinfo.CoreRanking{},
			expectError:  true,
		},
		{
			name: "Skip - Not Enough Cores",
			vmid: 104,
			pid:  12349,
			config: &proxmox.VmConfig{
				Cores:   4, // Request 4, have 3
				Sockets: 1,
			},
			mockRankings: rankings,
			expectedRes:  "",
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
			mockRankings: rankings,
			mockSysErr:   errors.New("sys error"),
			expectError:  false,
			expectedRes:  "1",
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
				cpuInfo:   mockCpu,
				sys:       mockSys,
				lastIndex: 0,
				config:    &config.Config{Rounds: 1, Iterations: 1},
			}

			// Setup Ranking Mock
			if tt.mockRankingErr != nil {
				mockCpu.On("GetCoreRanking").Return(nil, tt.mockRankingErr)
			} else {
				mockCpu.On("GetCoreRanking").Return(tt.mockRankings, nil)
			}

			// Setup Sys Mock
			if tt.setupMockSys != nil {
				tt.setupMockSys(mockSys)
			}

			res, err := p.ApplyAffinity(tt.vmid, tt.pid, tt.config)

			if tt.expectError {
				// xxx
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRes, res)
			}

			mockCpu.AssertExpectations(t)
			mockSys.AssertExpectations(t)
		})
	}
}
