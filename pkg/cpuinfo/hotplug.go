package cpuinfo

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"golang.org/x/sys/unix"
)

const (
	// Buffers prevent blocking if bursts occur
	EventBufferSize = 100
	JobBufferSize   = 10
)

var cpuNameRegexp = regexp.MustCompile(`cpu[0-9]+`)

type CPUAction int

const (
	ActionAdd CPUAction = iota
	ActionRemove
)

func (a CPUAction) String() string {
	switch a {
	case ActionAdd:
		return "add"
	case ActionRemove:
		return "remove"
	}
	return "unknown"
}

type CPUEvent struct {
	CPU    string
	Action CPUAction
}

// HotplugController defines the interface for managing the hotplug watchdog.
type HotplugController interface {
	StartWatchdog() error
	StopWatchdog() error
}

// Hotplug holds the CPUInfo provider and manages the watchdog.
type Hotplug struct {
	cpuInfo   Provider
	cfg       *config.Config
	netlinkFD int
	reactor   *hotplugReactor
	logger    *slog.Logger
}

// NewHotplug creates a new Hotplug instance.
func NewHotplug(cpuInfo Provider, cfg *config.Config) HotplugController {
	return &Hotplug{
		cpuInfo: cpuInfo,
		cfg:     cfg,
		logger:  slog.Default(),
	}
}

// Start starts the hotplug watchdog.
func (h *Hotplug) StartWatchdog() error {
	h.logger.Info("[cpu-hotplug] Starting watchdog")

	h.reactor = newHotplugReactor(config.CPUHotplugBatchWindow, func(batch []string) {
		h.logger.Info("[cpu-hotplug] Event detected - recalculating ranking", "batch_size", len(batch))
		if err := h.cpuInfo.CalculateRanking(h.cfg.Rounds, h.cfg.Iterations, config.MaxCalculationRankingDuration); err != nil {
			h.logger.Error("[cpu-hotplug] Failed to recalculate ranking after hotplug", "error", err)
		}
	}, h.logger)
	h.reactor.start()

	return h.startNetlink()
}

// Stop the hotplug watchdog.
func (h *Hotplug) StopWatchdog() error {
	h.logger.Info("[cpu-hotplug] Stopping watchdog")
	if h.reactor != nil {
		h.reactor.stop()
	}
	return h.stopNetlink()
}

func (h *Hotplug) startNetlink() error {
	h.logger.Info("[cpu-hotplug] Starting Netlink listener")

	// Establish a Netlink socket to listen for kernel uevents (hotplug), similar to udev.

	// AF_NETLINK: The socket family for communicating between kernel and user space.
	// NETLINK_KOBJECT_UEVENT: The specific protocol for kernel object events (hotplug).
	// This allows us to receive notifications when hardware (like CPUs) is added or removed.
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_KOBJECT_UEVENT)
	if err != nil {
		return fmt.Errorf("failed to create netlink socket: %w", err)
	}

	// Bind to the Netlink socket.
	// Groups: 1 - This is the multicast group for kernel uevents. We subscribe to this to hear broadcasts.
	// Pid: 0    - Setting PID to 0 tells the kernel to assign a unique Port ID to this socket automatically.
	addr := &unix.SockaddrNetlink{Family: unix.AF_NETLINK, Groups: 1, Pid: 0}
	if err := unix.Bind(fd, addr); err != nil {
		_ = unix.Close(fd)
		return fmt.Errorf("failed to bind netlink socket: %w", err)
	}

	h.netlinkFD = fd

	go func() {
		buf := make([]byte, 4096)
		for {
			n, _, err := unix.Recvfrom(fd, buf, 0)
			if err != nil {
				h.logger.Debug("[cpu-hotplug] Netlink socket read error (stopping?)", "error", err)
				return
			}
			msg := string(buf[:n])

			if strings.Contains(msg, "SUBSYSTEM=cpu") {
				var action CPUAction
				if strings.Contains(msg, "ACTION=add") {
					action = ActionAdd
				} else if strings.Contains(msg, "ACTION=remove") {
					action = ActionRemove
				} else {
					continue
				}

				h.reactor.ingest(CPUEvent{CPU: cpuNameRegexp.FindString(msg), Action: action})
			}
		}
	}()

	return nil
}

func (h *Hotplug) stopNetlink() error {
	h.logger.Info("[cpu-hotplug] Stopping Netlink listener")
	if h.netlinkFD != 0 {
		err := unix.Close(h.netlinkFD)
		h.netlinkFD = 0
		return err
	}
	return nil
}

// hotplugReactor handles the buffering and dispatching of events.
// It is separated from Hotplug to allow unit testing of the batching logic.
type hotplugReactor struct {
	events   chan CPUEvent
	jobs     chan []string
	stopChan chan struct{}
	window   time.Duration
	handler  func([]string)
	logger   *slog.Logger
}

func newHotplugReactor(window time.Duration, handler func([]string), logger *slog.Logger) *hotplugReactor {
	return &hotplugReactor{
		events:   make(chan CPUEvent, EventBufferSize),
		jobs:     make(chan []string, JobBufferSize),
		stopChan: make(chan struct{}),
		window:   window,
		handler:  handler,
		logger:   logger,
	}
}

func (r *hotplugReactor) start() {
	go r.processBatches()
	go r.workerLogic()
}

func (r *hotplugReactor) stop() {
	close(r.stopChan)
}

func (r *hotplugReactor) ingest(evt CPUEvent) {
	select {
	case r.events <- evt:
	default:
		r.logger.Warn("[hotplug-reactor] Event buffer full! Dropping event.")
	}
}

func (r *hotplugReactor) processBatches() {
	var batch []string
	timer := time.NewTimer(r.window)
	if !timer.Stop() {
		<-timer.C
	}
	timerRunning := false

	for {
		select {
		case <-r.stopChan:
			return
		case evt := <-r.events:
			batch = append(batch, evt.CPU)

			// Extend window
			if timerRunning {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}
			timer.Reset(r.window)
			timerRunning = true
			r.logger.Debug("[hotplug-reactor] Buffering hotplug event", "cpu", evt.CPU, "action", evt.Action, "batch_size", len(batch))

		case <-timer.C:
			timerRunning = false
			if len(batch) > 0 {
				job := batch
				batch = nil

				select {
				case r.jobs <- job:
					r.logger.Info("[hotplug-reactor] Batch sent to worker", "events", len(job))
				default:
					r.logger.Error("[hotplug-reactor] Job Queue full! Worker is too slow.")
				}
			}
		}
	}
}

func (r *hotplugReactor) workerLogic() {
	for {
		select {
		case <-r.stopChan:
			return
		case batch := <-r.jobs:
			r.handler(batch)
		}
	}
}
