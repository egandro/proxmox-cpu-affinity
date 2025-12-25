package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage proxmox-cpu-affinity hookscripts",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat("/etc/pve"); os.IsNotExist(err) {
				return fmt.Errorf("this tool must be run on a Proxmox VE host (/etc/pve not found)")
			}
			return nil
		},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newWebhookStatusCmd())
	cmd.AddCommand(newEnableCmd())
	cmd.AddCommand(newDisableCmd())
	cmd.AddCommand(newEnableAllCmd())
	cmd.AddCommand(newDisableAllCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all VMs and their hook status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%-8s %-12s %-30s\n", "VMID", "HOOK-STATUS", "NOTES")
			fmt.Printf("%-8s %-12s %-30s\n", "----", "-----------", "-----")
			for _, vmid := range getAllVMIDs() {
				printVMStatusRow(vmid)
			}
		},
	}
}

func newWebhookStatusCmd() *cobra.Command {
	return &cobra.Command{
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
			if err := exec.Command("/usr/sbin/qm", "config", strconv.FormatUint(vmid, 10)).Run(); err != nil {
				fmt.Printf("Error: VM %d not found.\n", vmid)
				os.Exit(1)
			}
			fmt.Printf("%-8s %-12s %-30s\n", "VMID", "HOOK-STATUS", "NOTES")
			fmt.Printf("%-8s %-12s %-30s\n", "----", "-----------", "-----")
			printVMStatusRow(vmid)
		},
	}
}

func newEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <vmid> [storage]",
		Short: "Enable hook for specific VM (default storage: local)",
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
			storage := "local"
			if len(args) > 1 {
				storage = args[1]
			}

			if isHAVM(vmid) {
				fmt.Printf("Error: VM %d is managed by HA. Cannot modify hookscript manually.\n", vmid)
				os.Exit(1)
			}
			if hasAffinitySet(vmid) {
				fmt.Printf("Error: VM %d has manual CPU affinity set. Cannot modify hookscript.\n", vmid)
				os.Exit(1)
			}
			enableVM(vmid, storage)
		},
	}
}

func newDisableCmd() *cobra.Command {
	return &cobra.Command{
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
			disableVM(vmid)
		},
	}
}

func newEnableAllCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "enable-all [storage]",
		Short: "Enable hook for ALL VMs (default storage: local)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			storage := "local"
			if len(args) > 0 {
				storage = args[0]
			}

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
				if strings.Contains(vmConf, "hookscript:") {
					if !strings.Contains(vmConf, hookscriptFile) {
						if !force {
							fmt.Printf("Skipping VM %d (other hookscript set)...\n", vmid)
							continue
						}
					}
				}
				enableVM(vmid, storage)
			}
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force enable even if other hookscripts are present")
	return cmd
}

func newDisableAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable-all",
		Short: "Disable hook for ALL VMs",
		Run: func(cmd *cobra.Command, args []string) {
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
				if !strings.Contains(vmConf, "hookscript: ") || !strings.Contains(vmConf, hookscriptFile) {
					continue
				}
				disableVM(vmid)
			}
		},
	}
}

// Helpers
func getHookPath(storage string) string {
	return fmt.Sprintf("%s:snippets/%s", storage, hookscriptFile)
}

func isValidStorage(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-', c == '_', c == '.':
		default:
			return false
		}
	}
	return true
}

func enableVM(vmid uint64, storage string) {
	if !isValidStorage(storage) {
		fmt.Printf("Error: Invalid storage %s\n", storage)
		return
	}
	hookPath := getHookPath(storage)
	fmt.Printf("Enabling proxmox-cpu-affinity hook for VM %d (storage: %s)...\n", vmid, storage)
	// #nosec G204 -- vmid is uint64, hookPath is validated
	if err := exec.Command("/usr/sbin/qm", "set", strconv.FormatUint(vmid, 10), "--hookscript", hookPath).Run(); err != nil {
		fmt.Printf("Error enabling hook for VM %d: %v\n", vmid, err)
	}
}

func disableVM(vmid uint64) {
	fmt.Printf("Disabling proxmox-cpu-affinity hook for VM %d...\n", vmid)
	// #nosec G204 -- vmid is uint64
	if err := exec.Command("/usr/sbin/qm", "set", strconv.FormatUint(vmid, 10), "--delete", "hookscript").Run(); err != nil {
		fmt.Printf("Error disabling hook for VM %d: %v\n", vmid, err)
	}
}

func getAllVMIDs() []uint64 {
	out, _ := exec.Command("/usr/sbin/qm", "list").Output()
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
	return vmids
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

var (
	haConfigCache  string
	haConfigLoaded bool
)

func isHAVM(vmid uint64) bool {
	if !haConfigLoaded {
		out, _ := exec.Command("/usr/sbin/ha-manager", "config").Output()
		haConfigCache = string(out)
		haConfigLoaded = true
	}
	// Simple check for "vm: <vmid>"
	return strings.Contains(haConfigCache, fmt.Sprintf("vm: %d", vmid))
}

func hasAffinitySet(vmid uint64) bool {
	out, _ := exec.Command("/usr/sbin/qm", "config", strconv.FormatUint(vmid, 10)).Output() // #nosec G204 -- vmid is uint64
	return strings.Contains(string(out), "affinity:")
}

func getVMConfig(vmid uint64) string {
	out, _ := exec.Command("/usr/sbin/qm", "config", strconv.FormatUint(vmid, 10)).Output() // #nosec G204 -- vmid is uint64
	return string(out)
}

func printVMStatusRow(vmid uint64) {
	status := "DISABLED"
	notes := ""
	vmConf := getVMConfig(vmid)

	if strings.Contains(vmConf, "template: 1") {
		notes = "VM Template"
	}

	if isHAVM(vmid) {
		status = "SKIPPED"
		if notes != "" {
			notes += ", "
		}
		notes += "HA Managed"
	} else if strings.Contains(vmConf, "affinity:") {
		status = "SKIPPED"
		if notes != "" {
			notes += ", "
		}
		notes += "Manual Affinity Set"
	} else if strings.Contains(vmConf, "hookscript: ") && strings.Contains(vmConf, hookscriptFile) {
		status = "ENABLED"
	}

	fmt.Printf("%-8d %-12s %-30s\n", vmid, status, notes)
}
