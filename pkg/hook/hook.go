package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	if h.Config.SocketPingOnPreStart {
		if err := h.callService("ping", vmid); err != nil {
			_, _ = fmt.Fprintf(h.Output, "Warning: Service not reachable: %v\n", err)
		}
	}
	return nil
}

// OnPostStart is executed after the guest successfully started.
func (h *handler) OnPostStart(vmid int) error {
	if err := h.callService("update-affinity", vmid); err != nil {
		_, _ = fmt.Fprintf(h.Output, "Error calling service: %v\n", err)
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

func (h *handler) callService(command string, vmid int) error {
	var err error
	for i := 0; i <= h.Config.SocketRetry; i++ {
		if i > 0 {
			time.Sleep(time.Duration(h.Config.SocketSleep) * time.Second)
		}

		err = func() error {
			timeout := time.Duration(h.Config.SocketTimeout) * time.Second
			conn, err := net.DialTimeout("unix", h.Config.SocketFile, timeout)
			if err != nil {
				return err
			}
			defer func() {
				_ = conn.Close()
			}()

			_ = conn.SetDeadline(time.Now().Add(timeout))

			req := map[string]interface{}{
				"command": command,
				"vmid":    vmid,
			}
			if err := json.NewEncoder(conn).Encode(req); err != nil {
				return err
			}

			var resp struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			}
			if err := json.NewDecoder(conn).Decode(&resp); err != nil {
				return err
			}

			if resp.Status != "ok" {
				return fmt.Errorf("service returned error: %s", resp.Error)
			}
			return nil
		}()

		if err == nil {
			return nil
		}
	}
	return err
}
