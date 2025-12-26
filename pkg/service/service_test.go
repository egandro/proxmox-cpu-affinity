package service

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
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

func (m *MockScheduler) UpdateAffinity(ctx context.Context, vmid int) (interface{}, error) {
	args := m.Called(ctx, vmid)
	return args.Get(0), args.Error(1)
}

// MockCpuInfo is a mock implementation of cpuinfo.Provider.
type MockCpuInfo struct {
	mock.Mock
}

func (m *MockCpuInfo) Update(rounds int, iterations int, onProgress func(int, int)) error {
	args := m.Called(rounds, iterations, onProgress)
	return args.Error(0)
}

func (m *MockCpuInfo) CalculateRanking(rounds, iterations int, timeout time.Duration) error {
	args := m.Called(rounds, iterations, timeout)
	return args.Error(0)
}

func (m *MockCpuInfo) GetCoreRanking() ([]cpuinfo.CoreRanking, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpuinfo.CoreRanking), args.Error(1)
}

func (m *MockCpuInfo) SelectCores(requestedCores int) ([]int, error) {
	args := m.Called(requestedCores)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockCpuInfo) DetectTopology() ([]cpuinfo.CoreInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]cpuinfo.CoreInfo), args.Error(1)
}

func setupTestService(t *testing.T) (*MockScheduler, *MockCpuInfo, string) {
	// 1. Create a temporary socket path
	tmpDir := t.TempDir()
	// t.TempDir() creates a unique directory for each test, so a fixed filename is safe.
	socketPath := filepath.Join(tmpDir, "pca-test.sock")

	// 2. Setup Mock Scheduler
	mockSched := new(MockScheduler)

	// 3. Setup Mock CpuInfo
	mockCpuInfo := new(MockCpuInfo)

	// 4. Create and Start Service
	svc := New(t.Context(), socketPath, mockSched, mockCpuInfo)

	// Start in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- svc.Start()
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	t.Cleanup(func() {
		// Use context.Background() because t.Context() is cancelled during cleanup.
		// We need a fresh context to allow the shutdown to complete gracefully if needed.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := svc.Shutdown(ctx)
		assert.NoError(t, err)

		select {
		case err := <-errChan:
			assert.NoError(t, err)
		case <-time.After(1 * time.Second):
			t.Fatal("Service did not exit after shutdown")
		}
		_ = os.Remove(socketPath)
	})

	return mockSched, mockCpuInfo, socketPath
}

func TestService_UpdateAffinity(t *testing.T) {
	mockSched, _, socketPath := setupTestService(t)

	expectedResult := map[string]interface{}{
		"vmid":   100,
		"action": "start",
		"mocked": true,
	}
	mockSched.On("UpdateAffinity", mock.Anything, 100).Return(expectedResult, nil)

	// Dial
	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	// Send Request
	req := Request{Command: "update-affinity", VMID: 100}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	// Read Response
	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "ok", resp.Status)
	mockSched.AssertExpectations(t)
}

func TestService_UpdateAffinity_InvalidID(t *testing.T) {
	// We don't need a full server start for this if we could test handler directly,
	// but integration testing the router is safer.
	// For brevity, relying on the previous test structure is fine.
	// This test is just a placeholder to show where edge cases would go.
}

func TestService_HandleConnection_EOF(t *testing.T) {
	_, _, socketPath := setupTestService(t)

	// Dial
	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)

	// Close immediately to trigger EOF on server side decode
	_ = conn.Close()

	// Allow some time for server to process
	time.Sleep(50 * time.Millisecond)
}
