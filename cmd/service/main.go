package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/logger"
	"github.com/egandro/proxmox-cpu-affinity/pkg/scheduler"
	"github.com/egandro/proxmox-cpu-affinity/pkg/service"
)

func main() {
	configFile := flag.String("config", config.DefaultConfigFilename, "Path to config file")
	hostFlag := flag.String("host", "", "HTTP service host")
	portFlag := flag.Int("port", 0, "HTTP service port")
	logFileFlag := flag.String("log-file", "", "Path to log file")
	logLevelFlag := flag.String("log-level", "", "Log level (debug, info, notice, warn, error)")
	toStdout := flag.Bool("stdout", false, "Log to stdout")

	flag.Parse()

	cfg := config.Load(*configFile)

	// Override config with flags if provided
	if *hostFlag != "" {
		cfg.ServiceHost = *hostFlag
	}
	if *portFlag != 0 {
		cfg.ServicePort = *portFlag
	}
	if *logFileFlag != "" {
		cfg.LogFile = *logFileFlag
	}
	if *logLevelFlag != "" {
		cfg.LogLevel = *logLevelFlag
	}

	var logF *os.File
	var output io.Writer = os.Stdout

	if !*toStdout {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file %s: %v. Logging to stdout.\n", cfg.LogFile, err)
		} else {
			logF = f
			output = f
		}
	}

	// Configure slog level
	level, err := logger.ParseLevel(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v, defaulting to INFO\n", err)
	}

	handler := &logger.SimpleHandler{Output: output, Level: level}
	slog.SetDefault(slog.New(handler))

	sched, err := scheduler.New()
	if err != nil {
		slog.Error("Failed to initialize scheduler", "error", err)
		os.Exit(1)
	}
	s := service.New(cfg.ServiceHost, cfg.ServicePort, sched)

	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("Service failed", "error", err)
			os.Exit(1)
		}
	}()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigChan {
		switch sig {
		case syscall.SIGHUP:
			if logF != nil {
				newF, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
				if err == nil {
					_ = logF.Close()
					logF = newF

					// Re-create slog handler with new file
					handler := &logger.SimpleHandler{Output: logF, Level: level}
					slog.SetDefault(slog.New(handler))

					slog.Info("Log file rotated")
				} else {
					slog.Error("Failed to rotate log", "error", err)
				}
			}
		case syscall.SIGINT, syscall.SIGTERM:
			slog.Info("Shutting down service...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				slog.Error("Shutdown error", "error", err)
			}
			return
		}
	}
}
