package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/spf13/cobra"
)

const (
	systemPS      = "/usr/bin/ps"
	systemTaskSet = "/usr/bin/taskset"
	systemPGrep   = "/usr/bin/pgrep"
)

type PSInfo struct {
	VMID       uint64     `json:"vmid"`
	PID        uint64     `json:"pid,omitempty"`
	HookStatus string     `json:"hook_status"`
	Affinity   string     `json:"affinity,omitempty"`
	Threads    []PSThread `json:"threads,omitempty"`
	Children   []PSChild  `json:"children,omitempty"`
	Error      string     `json:"error,omitempty"`
}

type PSThread struct {
	TID     string `json:"tid"`
	PSR     string `json:"psr"`
	Command string `json:"command"`
}

type PSChild struct {
	PID     string `json:"pid"`
	PSR     string `json:"psr"`
	Command string `json:"command"`
}

func newPSCmd() *cobra.Command {
	var verbose bool
	var jsonOutput bool
	var quiet bool
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

			var vmids []uint64
			var explicit bool

			if len(args) > 0 {
				vmid, _ := strconv.ParseUint(args[0], 10, 64)
				vmids = []uint64{vmid}
				explicit = true
			} else {
				files, _ := filepath.Glob(filepath.Join(config.ConstantQemuServerPidDir, "*.pid"))
				if len(files) == 0 {
					fmt.Println("No running VMs found.")
					return
				}
				for _, f := range files {
					vmidStr := strings.TrimSuffix(filepath.Base(f), ".pid")
					if vmid, err := strconv.ParseUint(vmidStr, 10, 64); err == nil {
						vmids = append(vmids, vmid)
					}
				}
				sort.Slice(vmids, func(i, j int) bool { return vmids[i] < vmids[j] })
			}

			var results []PSInfo

			var s *spinner.Spinner
			if !quiet {
				s = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
				s.Suffix = " Collecting process info..."
				s.Start()
			}

			for _, vmid := range vmids {
				if info := getVMProcessInfo(vmid, verbose, explicit); info != nil {
					results = append(results, *info)
				}
			}

			if s != nil {
				s.Stop()
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(results)
			} else {
				printPSHeader()
				for _, info := range results {
					printVMProcessInfo(info, verbose)
				}
			}
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed thread and child process affinity")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Disable progress spinner")
	return cmd
}

func printPSHeader() {
	fmt.Printf("%-8s %-10s %-12s %-20s\n", "VMID", "PID", "Hook-Status", "Affinity")
	fmt.Printf("%-8s %-10s %-12s %-20s\n", "----", "---", "-----------", "--------")
}

func getVMProcessInfo(vmid uint64, verbose bool, explicit bool) *PSInfo {
	pidFile := filepath.Join(config.ConstantQemuServerPidDir, fmt.Sprintf("%d.pid", vmid))
	pidBytes, err := os.ReadFile(pidFile) // #nosec G304 -- vmid is uint64, path is safe
	if err != nil {
		if explicit {
			return &PSInfo{VMID: vmid, Error: "VM is not running (PID file not found)"}
		}
		return nil
	}
	// Validate and convert PID to integer
	pid, err := strconv.ParseUint(strings.TrimSpace(string(pidBytes)), 10, 64)
	if err != nil {
		if explicit {
			return &PSInfo{VMID: vmid, Error: fmt.Sprintf("Invalid PID in file: %v", err)}
		}
		return nil
	}

	// Check hookscript
	hookStatus := "Disabled"
	out, _ := exec.Command(config.ConstantProxmoxQM, "config", strconv.FormatUint(vmid, 10)).Output() // #nosec G204 -- vmid is uint64
	if strings.Contains(string(out), "hookscript: ") && strings.Contains(string(out), config.ConstantHookScriptFilename) {
		hookStatus = "Enabled"
	}

	info := &PSInfo{VMID: vmid, PID: pid, HookStatus: hookStatus}

	// Check if process exists
	// #nosec G204 -- pid is uint64
	if err := exec.Command(systemPS, "-p", strconv.FormatUint(pid, 10)).Run(); err == nil {
		// taskset -cp "$pid"
		tsOut, _ := exec.Command(systemTaskSet, "-cp", strconv.FormatUint(pid, 10)).CombinedOutput() // #nosec G204 -- pid is uint64
		affinity := strings.TrimSpace(string(tsOut))
		if idx := strings.Index(affinity, ":"); idx != -1 {
			affinity = strings.TrimSpace(affinity[idx+1:])
		}
		info.Affinity = affinity

		if verbose {
			// ps -L -p "$pid" -o tid,psr,comm
			psOut, _ := exec.Command(systemPS, "-L", "-p", strconv.FormatUint(pid, 10), "-o", "tid,psr,comm").Output() // #nosec G204 -- pid is uint64
			lines := strings.Split(string(psOut), "\n")
			for i, line := range lines {
				if i == 0 || line == "" {
					continue
				}
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					info.Threads = append(info.Threads, PSThread{
						TID: fields[0], PSR: fields[1], Command: strings.Join(fields[2:], " "),
					})
				}
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
				// ps -p "$children" -o pid,psr,comm
				psChildOut, _ := exec.Command(systemPS, "-p", children, "-o", "pid,psr,comm").Output() // #nosec G204 -- children is validated
				cLines := strings.Split(string(psChildOut), "\n")
				for i, line := range cLines {
					if i == 0 || line == "" {
						continue
					}
					fields := strings.Fields(line)
					if len(fields) >= 3 {
						info.Children = append(info.Children, PSChild{
							PID: fields[0], PSR: fields[1], Command: strings.Join(fields[2:], " "),
						})
					}
				}
			}
		}
	} else if explicit {
		info.Error = "Process not running"
	} else {
		return nil
	}
	return info
}

func printVMProcessInfo(info PSInfo, verbose bool) {
	if info.Error != "" {
		fmt.Printf("Error: VM %d: %s\n", info.VMID, info.Error)
		return
	}
	fmt.Printf("%-8d %-10d %-12s %s\n", info.VMID, info.PID, info.HookStatus, info.Affinity)
	if verbose {
		if len(info.Threads) > 0 {
			fmt.Println("  Threads (TID PSR COMMAND):")
			for _, t := range info.Threads {
				fmt.Printf("    %s %s %s\n", t.TID, t.PSR, t.Command)
			}
		}
		if len(info.Children) > 0 {
			fmt.Println("  Child Processes (PID PSR COMMAND):")
			for _, c := range info.Children {
				fmt.Printf("    %s %s %s\n", c.PID, c.PSR, c.Command)
			}
		}
		fmt.Println("")
	}
}
