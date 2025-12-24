package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/scheduler"
)

// service represents the HTTP service.
type service struct {
	Host      string
	Port      int
	server    *http.Server
	scheduler scheduler.Scheduler
	cpuInfo   cpuinfo.Provider
}

// New creates a new service instance.
func New(host string, port int, sched scheduler.Scheduler, cpuInfo cpuinfo.Provider) *service {
	return &service{
		Host:      host,
		Port:      port,
		scheduler: sched,
		cpuInfo:   cpuInfo,
	}
}

// Start runs the HTTP server.
func (s *service) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/vmstarted/{vmid}", s.handleVmStarted)
	mux.HandleFunc("GET /api/ranking", s.handleGetCoreRanking)
	mux.HandleFunc("GET /api/ping", s.handlePing)

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	slog.Info("Starting HTTP service", "address", addr)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *service) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *service) handleVmStarted(w http.ResponseWriter, r *http.Request) {
	vmidStr := r.PathValue("vmid")
	vmid, err := strconv.Atoi(vmidStr)
	if err != nil {
		s.respond(w, http.StatusBadRequest, map[string]string{"error": "Invalid VMID"})
		return
	}
	result, err := s.scheduler.VmStarted(vmid)
	if err != nil {
		s.respond(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respond(w, http.StatusOK, result)
}

func (s *service) handleGetCoreRanking(w http.ResponseWriter, r *http.Request) {
	rankings, err := s.cpuInfo.GetCoreRanking()
	if err != nil {
		s.respond(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.respond(w, http.StatusOK, rankings)
}

func (s *service) handlePing(w http.ResponseWriter, r *http.Request) {
	slog.Debug("ping received")
	s.respond(w, http.StatusOK, map[string]string{"ping": "pong"})
}

func (s *service) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}
