package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/egandro/proxmox-cpu-affinity/pkg/svg"
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
	cmd.AddCommand(newCoreRankingCmd(&socketFile))
	cmd.AddCommand(newCoreRankingSummaryCmd(&socketFile))
	cmd.AddCommand(newCoreVMAffinityCmd(&socketFile))
	cmd.AddCommand(newSvgCmd(&socketFile))
	return cmd
}

func newPingCmd(socketFile *string) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Ping the service",
		Run: func(cmd *cobra.Command, args []string) {
			targetSocket := resolveSocketPath(*socketFile)
			resp, err := sendSocketRequest(targetSocket, SocketRequest{Command: "ping"})
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				fmt.Println("Hint: The proxmox-cpu-affinity-service might not be running or is currently starting.")
				os.Exit(1)
			}
			if resp.Status != "ok" {
				fmt.Printf("Error: %s\n", resp.Error)
				os.Exit(1)
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(resp.Data)
				return
			}

			fmt.Printf("%v\n", resp.Data)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func newCoreRankingCmd(socketFile *string) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "core-ranking",
		Short: "Get the current core ranking",
		Run: func(cmd *cobra.Command, args []string) {
			var rankings []cpuinfo.CoreRanking
			if err := fetchServiceData(*socketFile, "core-ranking", &rankings); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(rankings)
				return
			}

			printCoreRankings(rankings)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func printCoreRankings(rankings []cpuinfo.CoreRanking) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "Source\tNeighbor\tSocket\tCore\tLatency (ns)")
	_, _ = fmt.Fprintln(w, "------\t--------\t------\t----\t------------")

	for _, r := range rankings {
		for _, n := range r.Ranking {
			_, _ = fmt.Fprintf(w, "%d\t%d\t%d\t%d\t%.2f\n", r.CPU, n.CPU, n.Socket, n.Core, n.LatencyNS)
		}
	}
	_ = w.Flush()
}

func newSvgCmd(socketFile *string) *cobra.Command {
	var outputFile string
	var showAffinity bool

	cmd := &cobra.Command{
		Use:   "svg",
		Short: "Export current status as SVG",
		Run: func(cmd *cobra.Command, args []string) {
			var rankings []cpuinfo.CoreRanking
			if err := fetchServiceData(*socketFile, "core-ranking", &rankings); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			var stats cpuinfo.TopologyStats
			if err := fetchServiceData(*socketFile, "core-ranking-summary", &stats); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			var selections map[int][]int
			if err := fetchServiceData(*socketFile, "core-vm-affinity", &selections); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			cpuName := getCPUModelName(config.ConstantProcCpuInfo)
			mode := svg.ModeDefault
			if showAffinity {
				mode = svg.ModeAffinity
			}
			heatmap := svg.New(rankings, stats, selections, cpuName, mode)
			data, err := heatmap.Generate()
			if err != nil {
				fmt.Printf("Error generating SVG: %v\n", err)
				os.Exit(1)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(data), 0600); err != nil {
					fmt.Printf("Error writing output file: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println(data)
			}
		},
	}
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default is stdout)")
	cmd.Flags().BoolVar(&showAffinity, "affinity", false, "Show current VMs affinity in SVG")
	return cmd
}

func newCoreRankingSummaryCmd(socketFile *string) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "core-ranking-summary",
		Short: "Get the core ranking summary",
		Run: func(cmd *cobra.Command, args []string) {
			var stats cpuinfo.TopologyStats
			if err := fetchServiceData(*socketFile, "core-ranking-summary", &stats); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(stats)
				return
			}

			printCoreRankingSummary(stats)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func printCoreRankingSummary(stats cpuinfo.TopologyStats) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintf(w, "CPU Count:\t%d\n", stats.CPUCount)
	_, _ = fmt.Fprintf(w, "Socket Count:\t%d\n", stats.SocketCount)
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Metric\tLatency (ns)")
	_, _ = fmt.Fprintln(w, "------\t------------")
	_, _ = fmt.Fprintf(w, "Min (Best)\t%.2f\n", stats.MinLatencyNS)
	_, _ = fmt.Fprintf(w, "Max (Worst)\t%.2f\n", stats.MaxLatencyNS)
	_ = w.Flush()
}

func newCoreVMAffinityCmd(socketFile *string) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "core-vm-affinity",
		Short: "Get the current CPU affinity selections by VMID",
		Run: func(cmd *cobra.Command, args []string) {
			var selections map[int][]int
			if err := fetchServiceData(*socketFile, "core-vm-affinity", &selections); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(selections)
				return
			}

			printCoreVMAffinity(selections)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func printCoreVMAffinity(selections map[int][]int) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "VMID\tSelected CPUs")
	_, _ = fmt.Fprintln(w, "----\t-------------")

	// Sort keys
	var vmids []int
	for vmid := range selections {
		vmids = append(vmids, vmid)
	}
	sort.Ints(vmids)

	for _, vmid := range vmids {
		cpus := selections[vmid]
		// Convert ints to strings for joining
		var cpuStrs []string
		for _, cpu := range cpus {
			cpuStrs = append(cpuStrs, fmt.Sprintf("%d", cpu))
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\n", vmid, strings.Join(cpuStrs, ","))
	}
	_ = w.Flush()
}

func fetchServiceData(socketFile string, command string, target interface{}) error {
	targetSocket := resolveSocketPath(socketFile)
	resp, err := sendSocketRequest(targetSocket, SocketRequest{Command: command})
	if err != nil {
		return err
	}

	if resp.Status != "ok" {
		return fmt.Errorf("%s", resp.Error)
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return fmt.Errorf("processing response data: %w", err)
	}

	return json.Unmarshal(dataBytes, target)
}
