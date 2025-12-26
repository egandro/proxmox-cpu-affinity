package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/scheduler"
)

// Request represents the JSON request structure.
type Request struct {
	Command string `json:"command"`
	VMID    int    `json:"vmid"`
}

// Response represents the JSON response structure.
type Response struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// service represents the socket service.
type service struct {
	ctx        context.Context
	mu         sync.Mutex
	SocketPath string
	listener   net.Listener
	scheduler  scheduler.Scheduler
	cpuInfo    cpuinfo.Provider
}

// New creates a new service instance.
func New(ctx context.Context, socketPath string, sched scheduler.Scheduler, cpuInfo cpuinfo.Provider) *service {
	return &service{
		ctx:        ctx,
		SocketPath: socketPath,
		scheduler:  sched,
		cpuInfo:    cpuInfo,
	}
}

// Start runs the socket listener.
func (s *service) Start() error {
	// Remove existing socket if it exists
	if _, err := os.Stat(s.SocketPath); err == nil {
		if err := os.Remove(s.SocketPath); err != nil {
			return fmt.Errorf("failed to remove existing socket: %w", err)
		}
	}

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket %s: %w", s.SocketPath, err)
	}

	// Set restrictive permissions so only root can access it
	if err := os.Chmod(s.SocketPath, 0600); err != nil {
		_ = listener.Close()
		return fmt.Errorf("failed to chmod socket: %w", err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()
	slog.Info("Starting socket service", "socket", s.SocketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept error: %w", err)
		}
		go s.handleConnection(s.ctx, conn)
	}
}

// Shutdown gracefully shuts down the server.
func (s *service) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *service) handleConnection(ctx context.Context, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	// Set a deadline for the interaction
	_ = conn.SetDeadline(time.Now().Add(config.ConstantSocketTimeout))

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			//slog.Debug("Connection closed by client (EOF)")
			return
		}
		slog.Error("Failed to decode request", "error", err)
		resp := Response{
			Status: "error",
			Error:  fmt.Sprintf("failed to decode request: %v", err),
		}
		_ = json.NewEncoder(conn).Encode(resp)
		return
	}

	var resp Response
	switch req.Command {
	case "update-affinity":
		result, err := s.scheduler.UpdateAffinity(ctx, req.VMID)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "ok"
			resp.Data = result
		}
	case "ping":
		slog.Debug("ping received")
		resp.Status = "ok"
		resp.Data = "pong"
	case "core-ranking":
		ranking, err := s.cpuInfo.GetCoreRanking()
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "ok"
			resp.Data = ranking
		}
	case "core-ranking-summary":
		ranking, err := s.cpuInfo.GetCoreRanking()
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "ok"
			resp.Data = cpuinfo.SummarizeRankings(ranking)
		}
	default:
		resp.Status = "error"
		resp.Error = fmt.Sprintf("unknown command: %s", req.Command)
	}

	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}
