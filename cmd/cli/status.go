package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check the status of the service",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.Load(config.ConstantConfigFilename)

			host := cfg.ServiceHost
			if host == "" || host == "0.0.0.0" {
				host = "127.0.0.1"
			}
			url := fmt.Sprintf("http://%s:%d/api/ping", host, cfg.ServicePort)

			client := http.Client{
				Timeout: 2 * time.Second,
			}

			resp, err := client.Get(url)
			if err != nil {
				fmt.Printf("Error: Service is not reachable: %v\n", err)
				os.Exit(1)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("Service is running")
			} else {
				fmt.Printf("Error: Service returned status %s\n", resp.Status)
				os.Exit(1)
			}
		},
	}
}
