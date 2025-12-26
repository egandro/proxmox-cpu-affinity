package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/spf13/cobra"
)

const defaultStorage = "local"

func newHookscriptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hookscript",
		Short: "Manage proxmox-cpu-affinity hookscripts",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return ensureProxmoxHost()
		},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newHookscriptStatusCmd())
	cmd.AddCommand(newEnableCmd())
	cmd.AddCommand(newDisableCmd())
	cmd.AddCommand(newEnableAllCmd())
	cmd.AddCommand(newDisableAllCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	var jsonOutput bool
	var quiet bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all VMs and their hook status",
		Run: func(cmd *cobra.Command, args []string) {
			printVMList(getAllVMIDs(), jsonOutput, quiet)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Disable progress spinner")
	return cmd
}

func newHookscriptStatusCmd() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "status <vmid>",
		Short: "Show hook status for specific VM",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if !isNumeric(args[0]) {
				return fmt.Errorf("invalid VMID: %s", args[0])
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			vmid, _ := strconv.ParseUint(args[0], 10, 64)
			// #nosec G204 -- vmid is uint64
			if err := exec.Command(config.ConstantProxmoxQM, "config", strconv.FormatUint(vmid, 10)).Run(); err != nil {
				fmt.Printf("Error: VM %d not found.\n", vmid)
				os.Exit(1)
			}
			printVMList([]uint64{vmid}, jsonOutput, false)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func newEnableCmd() *cobra.Command {
	var dryRun bool
	var force bool
	cmd := &cobra.Command{
		Use:   "enable <vmid> [storage]",
		Short: fmt.Sprintf("Enable hook for specific VM (default storage: %s)", defaultStorage),
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(1, 2)(cmd, args); err != nil {
				return err
			}
			if !isNumeric(args[0]) {
				return fmt.Errorf("invalid VMID: %s", args[0])
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			vmid, _ := strconv.ParseUint(args[0], 10, 64)
			storage := defaultStorage
			if len(args) > 1 {
				storage = args[1]
			}

			if err := checkStorage(storage); err != nil {
				if !force {
					fmt.Printf("Error: %v (use --force to override)\n", err)
					os.Exit(1)
				}
				fmt.Printf("Warning: %v (proceeding due to --force)\n", err)
			}

			if isHAVM(vmid) {
				fmt.Printf("Error: VM %d is managed by HA. Cannot modify hookscript manually.\n", vmid)
				os.Exit(1)
			}
			if hasAffinitySet(vmid) {
				fmt.Printf("Error: VM %d has manual CPU affinity set. Cannot modify hookscript.\n", vmid)
				os.Exit(1)
			}
			updateVMHook(vmid, true, storage, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")
	cmd.Flags().BoolVar(&force, "force", false, "Force enable even if storage check fails")
	return cmd
}

func newDisableCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "disable <vmid>",
		Short: "Disable hook for specific VM",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if !isNumeric(args[0]) {
				return fmt.Errorf("invalid VMID: %s", args[0])
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			vmid, _ := strconv.ParseUint(args[0], 10, 64)
			if isHAVM(vmid) {
				fmt.Printf("Error: VM %d is managed by HA. Cannot modify hookscript manually.\n", vmid)
				os.Exit(1)
			}
			if hasAffinitySet(vmid) {
				fmt.Printf("Error: VM %d has manual CPU affinity set. Cannot modify hookscript.\n", vmid)
				os.Exit(1)
			}
			updateVMHook(vmid, false, "", dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")
	return cmd
}

func newEnableAllCmd() *cobra.Command {
	var force bool
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "enable-all [storage]",
		Short: fmt.Sprintf("Enable hook for ALL VMs (default storage: %s)", defaultStorage),
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			storage := defaultStorage
			if len(args) > 0 {
				storage = args[0]
			}

			if err := checkStorage(storage); err != nil {
				if !force {
					fmt.Printf("Error: %v (use --force to override)\n", err)
					os.Exit(1)
				}
				fmt.Printf("Warning: %v (proceeding due to --force)\n", err)
			}

			processAllVMs(true, storage, force, dryRun)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force enable even if other hookscripts are present or storage check fails")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")
	return cmd
}

func newDisableAllCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "disable-all",
		Short: "Disable hook for ALL VMs",
		Run: func(cmd *cobra.Command, args []string) {
			processAllVMs(false, "", false, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print commands without executing them")
	return cmd
}

func updateVMHook(vmid uint64, enable bool, storage string, dryRun bool) {
	if enable {
		if !isValidStorage(storage) {
			fmt.Printf("Error: Invalid storage %s\n", storage)
			return
		}
		hookPath := getHookPath(storage)

		if dryRun {
			fmt.Printf("[DryRun] Would execute: %s set %d --hookscript %s\n", config.ConstantProxmoxQM, vmid, hookPath)
			return
		}

		fmt.Printf("Enabling proxmox-cpu-affinity hook for VM %d (storage: %s)...\n", vmid, storage)
		// #nosec G204 -- vmid is uint64, hookPath is validated
		if err := exec.Command(config.ConstantProxmoxQM, "set", strconv.FormatUint(vmid, 10), "--hookscript", hookPath).Run(); err != nil {
			fmt.Printf("Error enabling hook for VM %d: %v\n", vmid, err)
		}
	} else {
		if dryRun {
			fmt.Printf("[DryRun] Would execute: %s set %d --delete hookscript\n", config.ConstantProxmoxQM, vmid)
			return
		}

		fmt.Printf("Disabling proxmox-cpu-affinity hook for VM %d...\n", vmid)
		// #nosec G204 -- vmid is uint64
		if err := exec.Command(config.ConstantProxmoxQM, "set", strconv.FormatUint(vmid, 10), "--delete", "hookscript").Run(); err != nil {
			fmt.Printf("Error disabling hook for VM %d: %v\n", vmid, err)
		}
	}
}

func processAllVMs(enable bool, storage string, force bool, dryRun bool) {
	for _, vmid := range getAllVMIDs() {
		if isHAVM(vmid) {
			fmt.Printf("Skipping HA-managed VM %d...\n", vmid)
			continue
		}
		vmConf := getVMConfig(vmid)
		if strings.Contains(vmConf, "template: 1") {
			fmt.Printf("Skipping VM Template %d...\n", vmid)
			continue
		}
		if strings.Contains(vmConf, "affinity:") {
			fmt.Printf("Skipping VM %d (manual affinity set)...\n", vmid)
			continue
		}

		if enable {
			if strings.Contains(vmConf, "hookscript:") {
				if !strings.Contains(vmConf, config.ConstantHookScriptFilename) {
					if !force {
						fmt.Printf("Skipping VM %d (other hookscript set)...\n", vmid)
						continue
					}
				}
			}
			updateVMHook(vmid, true, storage, dryRun)
		} else {
			if !strings.Contains(vmConf, "hookscript: ") || !strings.Contains(vmConf, config.ConstantHookScriptFilename) {
				continue
			}
			updateVMHook(vmid, false, "", dryRun)
		}
	}
}

func getAllVMIDs() []uint64 {
	out, _ := exec.Command(config.ConstantProxmoxQM, "list").Output() // #nosec G204 -- trusted path from config
	lines := strings.Split(string(out), "\n")
	var vmids []uint64
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			if id, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
				vmids = append(vmids, id)
			}
		}
	}
	sort.Slice(vmids, func(i, j int) bool { return vmids[i] < vmids[j] })
	return vmids
}

var (
	haConfigCache  string
	haConfigLoaded bool
)

func isHAVM(vmid uint64) bool {
	if !haConfigLoaded {
		out, _ := exec.Command(config.ConstantProxmoxHaManager, "config").Output() // #nosec G204 -- trusted path from config
		haConfigCache = string(out)
		haConfigLoaded = true
	}
	// Simple check for "vm: <vmid>"
	return strings.Contains(haConfigCache, fmt.Sprintf("vm: %d", vmid))
}

func hasAffinitySet(vmid uint64) bool {
	out, _ := exec.Command(config.ConstantProxmoxQM, "config", strconv.FormatUint(vmid, 10)).Output() // #nosec G204 -- vmid is uint64
	return strings.Contains(string(out), "affinity:")
}

func getVMConfig(vmid uint64) string {
	out, _ := exec.Command(config.ConstantProxmoxQM, "config", strconv.FormatUint(vmid, 10)).Output() // #nosec G204 -- vmid is uint64
	return string(out)
}

type HookStatusInfo struct {
	VMID   uint64 `json:"vmid"`
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

func printVMList(vmids []uint64, jsonOutput bool, quiet bool) {
	var list []HookStatusInfo

	var s *spinner.Spinner
	if !quiet {
		s = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
		s.Suffix = " Fetching VM status..."
		s.Start()
	}

	for _, vmid := range vmids {
		status := "Disabled"
		notes := ""
		vmConf := getVMConfig(vmid)

		if strings.Contains(vmConf, "template: 1") {
			notes = "VM Template"
		}

		if isHAVM(vmid) {
			status = "Skipped"
			if notes != "" {
				notes += ", "
			}
			notes += "HA Managed"
		} else if strings.Contains(vmConf, "affinity:") {
			status = "Skipped"
			if notes != "" {
				notes += ", "
			}
			notes += "Manual Affinity Set"
		} else if strings.Contains(vmConf, "hookscript: ") && strings.Contains(vmConf, config.ConstantHookScriptFilename) {
			status = "Enabled"
		}

		list = append(list, HookStatusInfo{
			VMID:   vmid,
			Status: status,
			Notes:  notes,
		})
	}

	if s != nil {
		s.Stop()
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(list)
		return
	}

	fmt.Printf("%-8s %-12s %-30s\n", "VMID", "Hook-Status", "Notes")
	fmt.Printf("%-8s %-12s %-30s\n", "----", "-----------", "-----")
	for _, item := range list {
		fmt.Printf("%-8d %-12s %-30s\n", item.VMID, item.Status, item.Notes)
	}
}

func checkStorage(storageName string) error {
	cmd := exec.Command("/usr/sbin/pvesm", "status")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute pvesm status: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		// Expected format: <Name> <Type> <Status> ...
		if len(fields) >= 3 && fields[0] == storageName {
			if fields[2] == "active" {
				return nil
			}
			return fmt.Errorf("storage '%s' is not active (status: %s)", storageName, fields[2])
		}
	}

	return fmt.Errorf("storage '%s' not found", storageName)
}
