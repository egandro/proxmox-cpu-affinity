package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/egandro/proxmox-cpu-affinity/pkg/executor"
	"github.com/spf13/cobra"
)

type ReassignResult struct {
	VMID   uint64 `json:"vmid"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func newReassignCmd() *cobra.Command {
	var all bool
	var dryRun bool
	var socketFile string

	cmd := &cobra.Command{
		Use:   "reassign [vmid]",
		Short: "Reassign CPU affinity for running VMs with enabled hooks",
		Args: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return fmt.Errorf("cannot specify both --all and a VMID")
			}
			if !all && len(args) == 0 {
				return fmt.Errorf("requires a VMID argument or --all flag")
			}
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

			ctx := cmd.Context()
			exec := &executor.DefaultExecutor{}
			targetSocket := resolveSocketPath(socketFile)

			var vmidsToProcess []uint64
			if len(args) == 1 {
				vmid, _ := strconv.ParseUint(args[0], 10, 64)
				vmidsToProcess = []uint64{vmid}
			} else {
				allRunningVMIDs, err := getAllRunningVMIDs(ctx, exec)
				if err != nil {
					fmt.Printf("Error: Failed to get running VMs: %v\n", err)
					os.Exit(1)
				}
				if len(allRunningVMIDs) == 0 {
					fmt.Println("No running VMs found.")
					return
				}
				vmidsToProcess = allRunningVMIDs
			}

			fmt.Printf("%-8s %-10s %s\n", "VMID", "Status", "Notes")
			fmt.Printf("%-8s %-10s %s\n", "----", "------", "-----")

			for _, vmid := range vmidsToProcess {
				res := ReassignResult{VMID: vmid}
				vmConfigOutput, err := getVMConfigOutput(ctx, exec, vmid)
				if err != nil {
					res.Status = "skipped"
					res.Error = fmt.Sprintf("failed to get VM config: %v", err)
					printReassignResult(res)
					continue
				}

				if !isProxmoxCPUAffinityHookEnabled(vmConfigOutput) {
					res.Status = "skipped"
					res.Error = "proxmox-cpu-affinity hook not enabled"
					printReassignResult(res)
					continue
				}

				if dryRun {
					res.Status = "dry-run"
					res.Error = "would reassign"
					printReassignResult(res)
					continue
				}

				// #nosec G115 -- VMID is always a positive integer within int range
				resp, err := sendSocketRequest(targetSocket, SocketRequest{Command: "update-affinity", VMID: int(vmid)})
				if err != nil {
					res.Status = "failed"
					res.Error = fmt.Sprintf("service call failed: %v", err)
					printReassignResult(res)
					continue
				}

				if resp.Status != "ok" {
					res.Status = "failed"
					res.Error = resp.Error
					printReassignResult(res)
					continue
				}
				res.Status = "success"
				printReassignResult(res)
			}
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "Reassign all running VMs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print actions without executing them")
	cmd.Flags().StringVar(&socketFile, "socket", "", "Path to unix socket")
	return cmd
}

func printReassignResult(res ReassignResult) {
	notes := ""
	if res.Error != "" {
		notes = res.Error
	}
	fmt.Printf("%-8d %-10s %s\n", res.VMID, res.Status, notes)
}
