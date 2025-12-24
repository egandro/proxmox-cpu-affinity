package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
)

func main() {
	summary := flag.Bool("summary", false, "Print summary instead of JSON")
	showProgress := flag.Bool("progress", false, "Show progress during calculation")
	flag.Parse()

	cfg := config.Load("")

	var onProgress func(int, int)
	if *showProgress {
		onProgress = func(round, total int) {
			fmt.Fprintf(os.Stderr, "\rMeasuring latency: Round %d/%d", round, total)
			if round == total {
				fmt.Fprintln(os.Stderr)
			}
		}
	}
	c := cpuinfo.New(onProgress)
	if err := c.Update(cfg.Rounds, cfg.Iterations); err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating rankings: %v\n", err)
		os.Exit(1)
	}
	rankings, err := c.GetCoreRanking()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating rankings: %v\n", err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	var output interface{} = rankings
	if *summary {
		output = cpuinfo.SummarizeRankings(rankings)
	}

	if err := enc.Encode(output); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
