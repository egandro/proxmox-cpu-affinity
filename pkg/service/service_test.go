package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
)

// MockScheduler is a mock implementation of scheduler.Scheduler using testify/mock.
type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) VmStarted(vmid int) (interface{}, error) {
	args := m.Called(vmid)
	return args.Get(0), args.Error(1)
}

// MockCpuInfo is a mock implementation of cpuinfo.Provider.
type MockCpuInfo struct {
	mock.Mock
}

func (m *MockCpuInfo) Update(rounds int, iterations int) error {
	args := m.Called(rounds, iterations)
	return args.Error(0)
}

func (m *MockCpuInfo) GetCoreRanking() ([]cpuinfo.CoreRanking, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpuinfo.CoreRanking), args.Error(1)
}

func (m *MockCpuInfo) DetectTopology() ([]cpuinfo.CoreInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpuinfo.CoreInfo), args.Error(1)
}

func setupTestService(t *testing.T) (*MockScheduler, *MockCpuInfo, string, func()) {
	// 1. Find a free port on localhost
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close() // Close it so the service can use it

	// 2. Setup Mock Scheduler
	mockSched := new(MockScheduler)

	// 3. Setup Mock CpuInfo
	mockCpuInfo := new(MockCpuInfo)

	// 4. Create and Start Service
	svc := New("127.0.0.1", port, mockSched, mockCpuInfo)

	// Start in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- svc.Start()
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	teardown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := svc.Shutdown(ctx)
		assert.NoError(t, err)

		select {
		case err := <-errChan:
			assert.Equal(t, http.ErrServerClosed, err)
		case <-time.After(1 * time.Second):
			t.Fatal("Service did not exit after shutdown")
		}
	}

	return mockSched, mockCpuInfo, baseURL, teardown
}

func TestService_VmStarted(t *testing.T) {
	mockSched, _, baseURL, teardown := setupTestService(t)
	defer teardown()

	expectedResult := map[string]interface{}{
		"vmid":   100,
		"action": "start",
		"mocked": true,
	}
	mockSched.On("VmStarted", 100).Return(expectedResult, nil)

	url := fmt.Sprintf("%s/api/vmstarted/100", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockSched.AssertExpectations(t)
}

func TestService_GetCoreRanking(t *testing.T) {
	_, mockCpuInfo, baseURL, teardown := setupTestService(t)
	defer teardown()

	expectedRankings := []cpuinfo.CoreRanking{
		{CPU: 0, Ranking: []cpuinfo.Neighbor{{CPU: 1, LatencyNS: 10}}},
	}
	mockCpuInfo.On("GetCoreRanking").Return(expectedRankings, nil)

	url := fmt.Sprintf("%s/api/ranking", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockCpuInfo.AssertExpectations(t)
}

func TestService_Ping(t *testing.T) {
	_, _, baseURL, teardown := setupTestService(t)
	defer teardown()

	url := fmt.Sprintf("%s/api/ping", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestService_VmStarted_InvalidID(t *testing.T) {
	_, _, baseURL, teardown := setupTestService(t)
	defer teardown()

	url := fmt.Sprintf("%s/api/vmstarted/not-a-number", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestService_VmStarted_Error(t *testing.T) {
	mockSched, _, baseURL, teardown := setupTestService(t)
	defer teardown()

	mockSched.On("VmStarted", 999).Return(nil, fmt.Errorf("VM not found"))

	url := fmt.Sprintf("%s/api/vmstarted/999", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockSched.AssertExpectations(t)
}

func TestService_GetCoreRanking_Error(t *testing.T) {
	_, mockCpuInfo, baseURL, teardown := setupTestService(t)
	defer teardown()

	mockCpuInfo.On("GetCoreRanking").Return(nil, fmt.Errorf("ranking failed"))

	url := fmt.Sprintf("%s/api/ranking", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	mockCpuInfo.AssertExpectations(t)
}

func TestService_ShutdownNilServer(t *testing.T) {
	svc := &service{}
	err := svc.Shutdown(context.Background())
	assert.NoError(t, err)
}
