package hook

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
)

// Phase represents the lifecycle phase of a VM.
type Phase string

const (
	PhasePreStart  Phase = "pre-start"
	PhasePostStart Phase = "post-start"
	PhasePreStop   Phase = "pre-stop"
	PhasePostStop  Phase = "post-stop"
)

// EventHandler defines the lifecycle events for a Proxmox VM.
type EventHandler interface {
	OnPreStart(vmid int) error
	OnPostStart(vmid int) error
	OnPreStop(vmid int) error
	OnPostStop(vmid int) error
}

// Hook defines the entry point for handling hooks.
type Hook interface {
	Handle(vmid int, phase string) error
}

// handler prints hook events to an output writer.
type handler struct {
	Output io.Writer
}

// hook handles the dispatching of hooks to an EventHandler.
type hook struct {
	Handler EventHandler
}

// New creates a new hook with the default event handler.
func New() Hook {
	return &hook{
		Handler: &handler{
			Output: os.Stdout,
		},
	}
}

// Handle executes the logic for a specific VM hook phase.
func (h *hook) Handle(vmid int, phase string) error {
	switch Phase(phase) {
	case PhasePreStart:
		return h.Handler.OnPreStart(vmid)
	case PhasePostStart:
		return h.Handler.OnPostStart(vmid)
	case PhasePreStop:
		return h.Handler.OnPreStop(vmid)
	case PhasePostStop:
		return h.Handler.OnPostStop(vmid)

	default:
		return fmt.Errorf("got unknown phase '%s'", phase)
	}
}

// OnPreStart is executed before the guest is started.
// Exiting with a code != 0 will abort the start.
func (h *handler) OnPreStart(vmid int) error {
	return nil
}

// OnPostStart is executed after the guest successfully started.
func (h *handler) OnPostStart(vmid int) error {
	cfg := config.Load("")
	url := fmt.Sprintf("http://%s:%d/api/vmstarted/%d", cfg.ServiceHost, cfg.ServicePort, vmid)
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		_, _ = fmt.Fprintf(h.Output, "Error calling vmstarted service: %v\n", err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(h.Output, "vmstarted service returned non-OK status: %s\n", resp.Status)
	}
	return nil
}

// OnPreStop is executed before stopping the guest via the API.
func (h *handler) OnPreStop(vmid int) error {
	return nil
}

// OnPostStop is executed after the guest stopped.
func (h *handler) OnPostStop(vmid int) error {
	return nil
}
