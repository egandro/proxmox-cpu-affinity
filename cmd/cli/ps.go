package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/spf13/cobra"
)

const (
	systemPS      = "/usr/bin/ps"
	systemTaskSet = "/usr/bin/taskset"
	systemPGrep   = "/usr/bin/pgrep"
)

func newPSCmd() *cobra.Command {
	var verbose bool
	cmd := &cobra.Command{
		Use:   "ps [vmid]",
		Short: "Show affinity information for VMs",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
			}
			if len(args) == 1 && !isNumeric(args[0]) {
				return fmt.Errorf("invalid VMID: %s", args[0])
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := ensureProxmoxHost(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if len(args) > 0 {
				vmid, _ := strconv.ParseUint(args[0], 10, 64)
				checkVM(vmid, verbose, true)
			} else {
				files, _ := filepath.Glob(filepath.Join(config.ConstantQemuServerPidDir, "*.pid"))
				if len(files) == 0 {
					fmt.Println("No running VMs found.")
					return
				}
				for _, f := range files {
					vmidStr := strings.TrimSuffix(filepath.Base(f), ".pid")
					if vmid, err := strconv.ParseUint(vmidStr, 10, 64); err == nil {
						checkVM(vmid, verbose, false)
					}
				}
			}
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed thread and child process affinity")
	return cmd
}

func checkVM(vmid uint64, verbose bool, explicit bool) {
	pidFile := filepath.Join(config.ConstantQemuServerPidDir, fmt.Sprintf("%d.pid", vmid))
	pidBytes, err := os.ReadFile(pidFile) // #nosec G304 -- vmid is uint64, path is safe
	if err != nil {
		if explicit {
			fmt.Printf("Error: VM %d is not running (PID file not found).\n", vmid)
		}
		return
	}
	// Validate and convert PID to integer
	pid, err := strconv.ParseUint(strings.TrimSpace(string(pidBytes)), 10, 64)
	if err != nil {
		return
	}

	// Check hookscript
	hookMsg := "    "
	out, _ := exec.Command(config.ConstantProxmoxQM, "config", strconv.FormatUint(vmid, 10)).Output() // #nosec G204 -- vmid is uint64
	if strings.Contains(string(out), "hookscript: ") && strings.Contains(string(out), config.ConstantHookScriptFilename) {
		hookMsg = " (*)"
	}

	// Check if process exists
	// #nosec G204 -- pid is uint64
	if err := exec.Command(systemPS, "-p", strconv.FormatUint(pid, 10)).Run(); err == nil {
		fmt.Printf("VM %-8d%s: ", vmid, hookMsg)

		// taskset -cp "$pid"
		tsOut, _ := exec.Command(systemTaskSet, "-cp", strconv.FormatUint(pid, 10)).CombinedOutput() // #nosec G204 -- pid is uint64
		affinity := strings.TrimSpace(string(tsOut))
		if idx := strings.Index(affinity, ":"); idx != -1 {
			affinity = strings.TrimSpace(affinity[idx+1:])
		}
		fmt.Printf("PID %-8d Affinity: %s\n", pid, affinity)

		if verbose {
			fmt.Println("  Threads (TID PSR COMMAND):")
			// ps -L -p "$pid" -o tid,psr,comm
			psOut, _ := exec.Command(systemPS, "-L", "-p", strconv.FormatUint(pid, 10), "-o", "tid,psr,comm").Output() // #nosec G204 -- pid is uint64
			lines := strings.Split(string(psOut), "\n")
			for i, line := range lines {
				if i == 0 || line == "" {
					continue
				}
				fmt.Printf("    %s\n", line)
			}

			// Child processes
			pgrepOut, _ := exec.Command(systemPGrep, "-P", strconv.FormatUint(pid, 10)).Output() // #nosec G204 -- pid is uint64
			children := strings.ReplaceAll(strings.TrimSpace(string(pgrepOut)), "\n", ",")

			validChildren := children != ""
			if validChildren {
				for _, r := range children {
					if (r < '0' || r > '9') && r != ',' {
						validChildren = false
						break
					}
				}
			}

			if validChildren {
				fmt.Println("  Child Processes (PID PSR COMMAND):")
				// ps -p "$children" -o pid,psr,comm
				psChildOut, _ := exec.Command(systemPS, "-p", children, "-o", "pid,psr,comm").Output() // #nosec G204 -- children is validated
				cLines := strings.Split(string(psChildOut), "\n")
				for i, line := range cLines {
					if i == 0 || line == "" {
						continue
					}
					fmt.Printf("    %s\n", line)
				}
			}
			fmt.Println("")
		}
	}
}
