package hook

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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
	Config *config.Config
}

// hook handles the dispatching of hooks to an EventHandler.
type hook struct {
	Handler EventHandler
}

// New creates a new hook with the default event handler.
func New() Hook {
	cfg := config.Load("")
	return &hook{
		Handler: &handler{
			Output: os.Stdout,
			Config: cfg,
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
	// Ping the server and delay the start of the VM.
	// The server might be in its calculation loop.
	// Performing this delay during pre-start ensures we are in an environment
	// where the system is not running many Qemu processes yet.
	if h.Config.WebhookPingOnPreStart {
		if err := h.callService("/api/ping"); err != nil {
			_, _ = fmt.Fprintf(h.Output, "Error server not ready: %v\n", err)
		}
	}
	return nil
}

// OnPostStart is executed after the guest successfully started.
func (h *handler) OnPostStart(vmid int) error {
	path := fmt.Sprintf("/api/vmstarted/%d", vmid)
	if err := h.callService(path); err != nil {
		_, _ = fmt.Fprintf(h.Output, "Error calling vmstarted service: %v\n", err)
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

func (h *handler) callService(apiPath string) error {
	url := fmt.Sprintf("http://%s:%d%s", h.Config.ServiceHost, h.Config.ServicePort, apiPath)

	var err error
	for i := 0; i <= h.Config.WebhookRetry; i++ {
		if i > 0 {
			time.Sleep(time.Duration(h.Config.WebhookSleep) * time.Second)
		}

		err = func() error {
			client := &http.Client{
				Timeout: time.Duration(h.Config.WebhookTimeout) * time.Second,
			}
			resp, reqErr := client.Get(url) // #nosec G107
			if reqErr != nil {
				return reqErr
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("service returned non-OK status: %s", resp.Status)
			}
			return nil
		}()
		if err == nil {
			return nil
		}
	}
	return err
}
