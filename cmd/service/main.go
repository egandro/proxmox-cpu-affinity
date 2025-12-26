package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/logger"
	"github.com/egandro/proxmox-cpu-affinity/pkg/scheduler"
	"github.com/egandro/proxmox-cpu-affinity/pkg/service"
)

func main() {
	configFile := flag.String("config", config.ConstantConfigFilename, "Path to config file")
	socketFlag := flag.String("socket", "", "Unix socket path")
	logFileFlag := flag.String("log-file", "", "Path to log file")
	logLevelFlag := flag.String("log-level", "", "Log level (debug, info, notice, warn, error)")
	toStdout := flag.Bool("stdout", false, "Log to stdout")
	disableCpuHotplugWatchdog := flag.Bool("disable-cpu-hotplug-watchdog", false, "Disable CPU hotplug watchdog")

	flag.Parse()

	cfg := config.Load(*configFile)

	// Override config with flags if provided
	if *socketFlag != "" {
		cfg.SocketFile = *socketFlag
	}
	if *logFileFlag != "" {
		cfg.LogFile = *logFileFlag
	}
	if *logLevelFlag != "" {
		cfg.LogLevel = *logLevelFlag
	}
	if *disableCpuHotplugWatchdog {
		cfg.CPUHotplugWatchdog = false
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
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level '%s': %v, defaulting to INFO\n", cfg.LogLevel, err)
		level = slog.LevelInfo
	}

	handler := &logger.SimpleHandler{Output: output, Level: level}
	slog.SetDefault(slog.New(handler))

	slog.Info("Proxmox CPU affinity service starting")

	cpuInfo := cpuinfo.New()

	if err := cpuInfo.CalculateRanking(cfg.Rounds, cfg.Iterations, config.ConstantMaxCalculationRankingDuration); err != nil {
		slog.Error("Failed to calculate ranking", "error", err)
		os.Exit(1)
	}

	var hotplugController cpuinfo.HotplugController
	if cfg.CPUHotplugWatchdog {
		hotplugController = cpuinfo.NewHotplug(cpuInfo, cfg)
		if err := hotplugController.StartWatchdog(); err != nil {
			slog.Warn("Failed to start CPU hotplug watchdog", "error", err)
		}
	}

	sched, err := scheduler.New(cfg, cpuInfo)
	if err != nil {
		slog.Error("Failed to initialize scheduler", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := service.New(ctx, cfg.SocketFile, sched, cpuInfo)

	go func() {
		if err := s.Start(); err != nil {
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
			cancel()
			if hotplugController != nil {
				if err := hotplugController.StopWatchdog(); err != nil {
					slog.Error("Failed to stop hotplug watchdog", "error", err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				slog.Error("Shutdown error", "error", err)
			}
			return
		}
	}
}
