package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var socketFile string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check the status of the service",
		Run: func(cmd *cobra.Command, args []string) {
			targetSocket := resolveSocketPath(socketFile)

			resp, err := sendSocketRequest(targetSocket, SocketRequest{Command: "ping"})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if resp.Status != "ok" || resp.Data != "pong" {
				fmt.Printf("Error: Service did not respond with pong (status=%s, data=%v, err=%s)\n", resp.Status, resp.Data, resp.Error)
				os.Exit(1)
			}
			fmt.Println("Service is running (pong received)")
		},
	}
	cmd.PersistentFlags().StringVar(&socketFile, "socket", "", "Path to unix socket")
	cmd.AddCommand(newPingCmd(&socketFile))
	return cmd
}

func newPingCmd(socketFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Ping the service",
		Run: func(cmd *cobra.Command, args []string) {
			targetSocket := resolveSocketPath(*socketFile)
			resp, err := sendSocketRequest(targetSocket, SocketRequest{Command: "ping"})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			if resp.Status != "ok" {
				fmt.Printf("Error: %s\n", resp.Error)
				os.Exit(1)
			}
			fmt.Printf("%v\n", resp.Data)
		},
	}
}
