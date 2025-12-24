package hook

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestHandler_OnPreStart(t *testing.T) {
	t.Run("Disabled", func(t *testing.T) {
		var buf bytes.Buffer
		h := &handler{
			Output: &buf,
			Config: &config.Config{WebhookPingOnPreStart: false},
		}

		err := h.OnPreStart(100)
		assert.NoError(t, err)
	})

	t.Run("Enabled_Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/ping" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		u, _ := url.Parse(ts.URL)
		host := u.Hostname()
		port, _ := strconv.Atoi(u.Port())

		var buf bytes.Buffer
		h := &handler{
			Output: &buf,
			Config: &config.Config{
				WebhookPingOnPreStart: true,
				ServiceHost:           host,
				ServicePort:           port,
				WebhookRetry:          0,
				WebhookSleep:          0,
			},
		}

		err := h.OnPreStart(100)
		assert.NoError(t, err)
		assert.Empty(t, buf.String())
	})

	t.Run("Enabled_Failure", func(t *testing.T) {
		var buf bytes.Buffer
		h := &handler{
			Output: &buf,
			Config: &config.Config{
				WebhookPingOnPreStart: true,
				ServiceHost:           "127.0.0.1",
				ServicePort:           0, // Invalid port
				WebhookRetry:          0,
				WebhookSleep:          0,
			},
		}

		err := h.OnPreStart(100)
		assert.NoError(t, err)
		assert.Contains(t, buf.String(), "Error server not ready")
	})
}

func TestHandler_OnPostStart(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{
		Output: &buf,
		Config: &config.Config{WebhookRetry: 0, WebhookSleep: 0},
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
