package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "proxmox-cpu-affinity-cli",
		Short: "CLI tool for Proxmox CPU Affinity",
	}

	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newCPUInfoCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newWebhookCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
