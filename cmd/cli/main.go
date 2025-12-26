package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Warning: This tool should be run as root. Some commands may fail.")
	}

	rootCmd := &cobra.Command{
		Use:   "proxmox-cpu-affinity",
		Short: "CLI tool for Proxmox CPU Affinity",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newCPUInfoCmd())
	rootCmd.AddCommand(newPSCmd())
	rootCmd.AddCommand(newHookscriptCmd())
	rootCmd.AddCommand(newHelloWorldCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
