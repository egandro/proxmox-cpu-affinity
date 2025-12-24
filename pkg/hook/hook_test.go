package hook

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockHTTPClient is a mock HTTP client for testing.
type mockHTTPClient struct {
	responses []*http.Response
	errors    []error
	callCount int
}

func (m *mockHTTPClient) Get(url string) (*http.Response, error) {
	idx := m.callCount
	m.callCount++
	if idx >= len(m.responses) {
		return nil, m.errors[len(m.errors)-1]
	}
	if idx < len(m.errors) && m.errors[idx] != nil {
		return nil, m.errors[idx]
	}
	return m.responses[idx], nil
}

func newMockHTTPClient(statusCode int) *mockHTTPClient {
	return &mockHTTPClient{
		responses: []*http.Response{
			{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			},
			{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
		},
	}
}

func TestHandler_OnPreStart(t *testing.T) {
	var buf bytes.Buffer
	h := &handler{Output: &buf}

	err := h.OnPreStart(100)
	assert.NoError(t, err)
}

func TestHandler_OnPostStart(t *testing.T) {
	var buf bytes.Buffer
	mockClient := newMockHTTPClient(http.StatusOK)
	h := &handler{Output: &buf, client: mockClient}

	err := h.OnPostStart(100)
	assert.NoError(t, err)
	// Should have called health check and vmstarted endpoint
	assert.GreaterOrEqual(t, mockClient.callCount, 2)
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
