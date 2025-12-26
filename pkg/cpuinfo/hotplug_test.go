package cpuinfo

import (
	"log/slog"
	"testing"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHotplugReactor_Batching(t *testing.T) {
	received := make(chan []string, 1)
	handler := func(batch []string) {
		received <- batch
	}

	// Short window for testing
	window := 50 * time.Millisecond
	reactor := newHotplugReactor(window, handler, slog.Default())
	reactor.start()
	defer reactor.stop()

	// Send events
	reactor.ingest(CPUEvent{CPU: "cpu1", Action: ActionAdd})
	reactor.ingest(CPUEvent{CPU: "cpu2", Action: ActionAdd})

	// Wait for batch
	select {
	case batch := <-received:
		assert.Equal(t, 2, len(batch))
		assert.Contains(t, batch, "cpu1")
		assert.Contains(t, batch, "cpu2")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for batch")
	}
}

func TestHotplugReactor_Debounce(t *testing.T) {
	received := make(chan []string, 1)
	handler := func(batch []string) {
		received <- batch
	}

	window := 50 * time.Millisecond
	reactor := newHotplugReactor(window, handler, slog.Default())
	reactor.start()
	defer reactor.stop()

	// Send events spaced out but within window
	reactor.ingest(CPUEvent{CPU: "cpu1", Action: ActionAdd})
	time.Sleep(20 * time.Millisecond)
	reactor.ingest(CPUEvent{CPU: "cpu2", Action: ActionAdd})
	time.Sleep(20 * time.Millisecond)
	reactor.ingest(CPUEvent{CPU: "cpu3", Action: ActionAdd})

	// Total time ~40ms, window 50ms. Timer should reset.
	// Batch should arrive ~50ms after last event.

	select {
	case batch := <-received:
		assert.Equal(t, 3, len(batch))
		assert.Contains(t, batch, "cpu1")
		assert.Contains(t, batch, "cpu2")
		assert.Contains(t, batch, "cpu3")
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for batch")
	}
}

func TestHotplugReactor_BufferFull(t *testing.T) {
	reactor := newHotplugReactor(time.Second, func([]string) {}, slog.Default())
	// Do NOT start reactor, so channel fills up.

	// Fill buffer
	for i := 0; i < EventBufferSize; i++ {
		reactor.ingest(CPUEvent{CPU: "cpu", Action: ActionAdd})
	}

	assert.Equal(t, EventBufferSize, len(reactor.events))

	// Try one more
	reactor.ingest(CPUEvent{CPU: "overflow", Action: ActionAdd})

	// Should still be full, no panic, extra event dropped
	assert.Equal(t, EventBufferSize, len(reactor.events))
}

// MockProvider for testing Hotplug integration
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) Update(rounds int, iterations int, onProgress func(int, int)) error {
	args := m.Called(rounds, iterations, onProgress)
	return args.Error(0)
}

func (m *MockProvider) GetCoreRanking() ([]CoreRanking, error) {
	args := m.Called()
	return args.Get(0).([]CoreRanking), args.Error(1)
}

func (m *MockProvider) SelectCPUs(vmid int, requestedCPUs int) ([]int, error) {
	args := m.Called(vmid, requestedCPUs)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockProvider) GetSelections() map[int][]int {
	args := m.Called()
	return args.Get(0).(map[int][]int)
}

func (m *MockProvider) CalculateRanking(rounds, iterations int, timeout time.Duration) error {
	args := m.Called(rounds, iterations, timeout)
	return args.Error(0)
}

func (m *MockProvider) DetectTopology() ([]CoreInfo, error) {
	args := m.Called()
	return args.Get(0).([]CoreInfo), args.Error(1)
}

func TestCPUAction_String(t *testing.T) {
	assert.Equal(t, "add", ActionAdd.String())
	assert.Equal(t, "remove", ActionRemove.String())
	assert.Equal(t, "unknown", CPUAction(999).String())
}

func TestNewHotplug(t *testing.T) {
	mockCPU := new(MockProvider)
	cfg := &config.Config{}
	hc := NewHotplug(mockCPU, cfg)
	assert.NotNil(t, hc)

	h, ok := hc.(*Hotplug)
	assert.True(t, ok)
	assert.Equal(t, mockCPU, h.cpuInfo)
	assert.Equal(t, cfg, h.cfg)
}

func TestHotplug_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode (waits for batch window)")
	}

	mockCPU := new(MockProvider)
	// Expect CalculateRanking to be called with config values
	mockCPU.On("CalculateRanking", 10, 100, config.ConstantMaxCalculationRankingDuration).Return(nil)

	cfg := &config.Config{Rounds: 10, Iterations: 100}
	hc := NewHotplug(mockCPU, cfg)
	h := hc.(*Hotplug)

	// Start Watchdog (ignore error as Netlink might fail on test env)
	_ = h.StartWatchdog()
	defer func() { _ = h.StopWatchdog() }()

	// Inject event directly into reactor
	h.reactor.ingest(CPUEvent{CPU: "cpu0", Action: ActionAdd})

	// Wait for BatchWindow + buffer
	time.Sleep(config.ConstantCPUHotplugBatchWindow + 100*time.Millisecond)

	mockCPU.AssertExpectations(t)
}
