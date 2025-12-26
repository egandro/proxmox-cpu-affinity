package hook

import (
	"bytes"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestHandler_OnPreStart(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{
		Output: &buf,
		Config: &config.Config{},
	}

	err := h.OnPreStart(100)
	assert.NoError(t, err)
}

func TestHandler_OnPostStart(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{
		Output: &buf,
		Config: &config.Config{},
	}

	err := h.OnPostStart(100)
	assert.NoError(t, err)
}

func TestHandler_OnPreStop(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{Output: &buf}

	err := h.OnPreStop(100)
	assert.NoError(t, err)
}

func TestHandler_OnPostStop(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{Output: &buf}

	err := h.OnPostStop(100)
	assert.NoError(t, err)
}

func TestHook_Handle_Unknown(t *testing.T) {
	// Test via the interface returned by New()
	h := New()
	err := h.Handle(100, "unknown-phase")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "got unknown phase")
}

// MockEventHandler is a mock implementation of EventHandler for testing dispatch logic.
type MockEventHandler struct {
	CalledPhase string
	CalledVMID  int
}

func (m *MockEventHandler) OnPreStart(vmid int) error {
	m.CalledPhase = string(PhasePreStart)
	m.CalledVMID = vmid
	return nil
}

func (m *MockEventHandler) OnPostStart(vmid int) error {
	m.CalledPhase = string(PhasePostStart)
	m.CalledVMID = vmid
	return nil
}

func (m *MockEventHandler) OnPreStop(vmid int) error {
	m.CalledPhase = string(PhasePreStop)
	m.CalledVMID = vmid
	return nil
}

func (m *MockEventHandler) OnPostStop(vmid int) error {
	m.CalledPhase = string(PhasePostStop)
	m.CalledVMID = vmid
	return nil
}

func TestHook_Handle_Dispatch(t *testing.T) {
	tests := []struct {
		phase string
		vmid  int
	}{
		{"pre-start", 101},
		{"post-start", 102},
		{"pre-stop", 103},
		{"post-stop", 104},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			mock := &MockEventHandler{}
			h := &hook{Handler: mock}

			err := h.Handle(tt.vmid, tt.phase)

			assert.NoError(t, err)
			assert.Equal(t, tt.phase, mock.CalledPhase)
			assert.Equal(t, tt.vmid, mock.CalledVMID)
		})
	}
}
