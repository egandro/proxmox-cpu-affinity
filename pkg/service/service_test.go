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

func (m *MockScheduler) Init() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockScheduler) VmStarted(vmid int) (interface{}, error) {
	args := m.Called(vmid)
	return args.Get(0), args.Error(1)
}

func (m *MockScheduler) GetCoreRanking() ([]cpuinfo.CoreRanking, error) {
	args := m.Called()
	return args.Get(0).([]cpuinfo.CoreRanking), args.Error(1)
}

func setupTestService(t *testing.T) (*MockScheduler, string, func()) {
	// 1. Find a free port on localhost
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close() // Close it so the service can use it

	// 2. Setup Mock Scheduler
	mockSched := new(MockScheduler)

	// 3. Create and Start Service
	svc := New("127.0.0.1", port, mockSched)

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

	return mockSched, baseURL, teardown
}

func TestService_VmStarted(t *testing.T) {
	mockSched, baseURL, teardown := setupTestService(t)
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
	mockSched, baseURL, teardown := setupTestService(t)
	defer teardown()

	expectedRankings := []cpuinfo.CoreRanking{
		{CPU: 0, Ranking: []cpuinfo.Neighbor{{CPU: 1, LatencyNS: 10}}},
	}
	mockSched.On("GetCoreRanking").Return(expectedRankings, nil)

	url := fmt.Sprintf("%s/api/ranking", baseURL)
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mockSched.AssertExpectations(t)
}

func TestService_VmStarted_InvalidID(t *testing.T) {
	// We don't need a full server start for this if we could test handler directly,
	// but integration testing the router is safer.
	// For brevity, relying on the previous test structure is fine.
	// This test is just a placeholder to show where edge cases would go.
}
