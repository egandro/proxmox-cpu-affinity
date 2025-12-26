package service

import (
	"context"
	"encoding/json"
	"fmt"
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

func (m *MockCpuInfo) SelectCPUs(vmid int, requestedCPUs int) ([]int, error) {
	args := m.Called(vmid, requestedCPUs)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockCpuInfo) GetSelections() map[int][]int {
	args := m.Called()
	return args.Get(0).(map[int][]int)
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
	_, _, socketPath := setupTestService(t)

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	// Send invalid JSON (string instead of int for vmid)
	_, err = conn.Write([]byte(`{"command": "update-affinity", "vmid": "not-a-number"}`))
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "error", resp.Status)
}

func TestService_UpdateAffinity_Error(t *testing.T) {
	mockSched, _, socketPath := setupTestService(t)

	mockSched.On("UpdateAffinity", mock.Anything, 999).Return(nil, fmt.Errorf("VM not found"))

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	req := Request{Command: "update-affinity", VMID: 999}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "error", resp.Status)
	mockSched.AssertExpectations(t)
}

func TestService_ShutdownNilServer(t *testing.T) {
	svc := &service{}
	err := svc.Shutdown(context.Background())
	assert.NoError(t, err)
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

func TestService_Ping(t *testing.T) {
	_, _, socketPath := setupTestService(t)

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	req := Request{Command: "ping"}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "pong", resp.Data)
}

func TestService_CoreRanking(t *testing.T) {
	_, mockCpuInfo, socketPath := setupTestService(t)

	expectedRanking := []cpuinfo.CoreRanking{
		{CPU: 0, Ranking: []cpuinfo.Neighbor{}},
		{CPU: 1, Ranking: []cpuinfo.Neighbor{}},
	}
	mockCpuInfo.On("GetCoreRanking").Return(expectedRanking, nil)

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	req := Request{Command: "core-ranking"}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "ok", resp.Status)

	// Verify Data matches expectedRanking
	dataBytes, err := json.Marshal(resp.Data)
	assert.NoError(t, err)
	expectedBytes, err := json.Marshal(expectedRanking)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedBytes), string(dataBytes))

	mockCpuInfo.AssertExpectations(t)
}

func TestService_CoreRankingSummary(t *testing.T) {
	_, mockCpuInfo, socketPath := setupTestService(t)

	expectedRanking := []cpuinfo.CoreRanking{
		{CPU: 0, Ranking: []cpuinfo.Neighbor{}},
		{CPU: 1, Ranking: []cpuinfo.Neighbor{}},
	}
	mockCpuInfo.On("GetCoreRanking").Return(expectedRanking, nil)

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	req := Request{Command: "core-ranking-summary"}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "ok", resp.Status)

	// Verify Data matches expected summary
	expectedSummary := cpuinfo.SummarizeRankings(expectedRanking)
	dataBytes, err := json.Marshal(resp.Data)
	assert.NoError(t, err)
	expectedBytes, err := json.Marshal(expectedSummary)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedBytes), string(dataBytes))

	mockCpuInfo.AssertExpectations(t)
}

func TestService_CoreVMAffinity(t *testing.T) {
	_, mockCpuInfo, socketPath := setupTestService(t)

	expectedSelections := map[int][]int{
		100: {0, 1},
		101: {2, 3},
	}
	mockCpuInfo.On("GetSelections").Return(expectedSelections)

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	req := Request{Command: "core-vm-affinity"}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "ok", resp.Status)

	// Verify Data matches expected selections
	dataBytes, err := json.Marshal(resp.Data)
	assert.NoError(t, err)
	expectedBytes, err := json.Marshal(expectedSelections)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expectedBytes), string(dataBytes))

	mockCpuInfo.AssertExpectations(t)
}

func TestService_UnknownCommand(t *testing.T) {
	_, _, socketPath := setupTestService(t)

	conn, err := net.Dial("unix", socketPath)
	assert.NoError(t, err)
	defer func() { _ = conn.Close() }()

	req := Request{Command: "unknown-cmd"}
	err = json.NewEncoder(conn).Encode(req)
	assert.NoError(t, err)

	var resp Response
	err = json.NewDecoder(conn).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "error", resp.Status)
	assert.Contains(t, resp.Error, "unknown command")
}
