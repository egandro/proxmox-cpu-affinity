package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var socketFile string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check the status of the service",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.Load(config.ConstantConfigFilename)

			targetSocket := cfg.SocketFile
			if socketFile != "" {
				targetSocket = socketFile
			}

			conn, err := net.DialTimeout("unix", targetSocket, 2*time.Second)
			if err != nil {
				fmt.Printf("Error: Service is not reachable: %v\n", err)
				os.Exit(1)
			}
			defer func() {
				_ = conn.Close()
			}()

			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

			req := map[string]interface{}{"command": "ping"}
			if err := json.NewEncoder(conn).Encode(req); err != nil {
				fmt.Printf("Error: Failed to send ping: %v\n", err)
				os.Exit(1)
			}

			var resp struct {
				Status string `json:"status"`
				Data   string `json:"data"`
			}
			if err := json.NewDecoder(conn).Decode(&resp); err != nil || resp.Status != "ok" || resp.Data != "pong" {
				fmt.Printf("Error: Service did not respond with pong (status=%s, data=%s, err=%v)\n", resp.Status, resp.Data, err)
				os.Exit(1)
			}
			fmt.Println("Service is running (pong received)")
		},
	}
	cmd.Flags().StringVar(&socketFile, "socket", "", "Path to unix socket")
	return cmd
}
