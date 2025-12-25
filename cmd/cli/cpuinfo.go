package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/spf13/cobra"
)

func newCPUInfoCmd() *cobra.Command {
	var verbose bool
	var summary bool
	var rounds int
	var iterations int

	// Load config to get defaults
	defaultCfg := config.Load(config.DefaultConfigFilename)

	cmd := &cobra.Command{
		Use:   "cpuinfo",
		Short: "Calculate and show CPU topology ranking",
		RunE: func(cmd *cobra.Command, args []string) error {
			var onProgress func(int, int)
			if verbose {
				onProgress = func(round, total int) {
					fmt.Fprintf(os.Stderr, "\rMeasuring latency: Round %d/%d", round, total)
					if round == total {
						fmt.Fprintln(os.Stderr)
					}
				}
				fmt.Fprintf(os.Stderr, "Starting calculation (Rounds: %d, Iterations: %d)...\n", rounds, iterations)
			}

			ci := cpuinfo.New(onProgress)

			if err := ci.Update(rounds, iterations); err != nil {
				return err
			}
			if verbose {
				fmt.Fprintln(os.Stderr, "Done.")
			}

			rankings, err := ci.GetCoreRanking()
			if err != nil {
				return err
			}

			var output interface{} = rankings
			if summary {
				output = cpuinfo.SummarizeRankings(rankings)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show progress during calculation")
	cmd.Flags().BoolVar(&summary, "summary", false, "Print summary instead of JSON")
	cmd.Flags().IntVar(&rounds, "rounds", defaultCfg.Rounds, "Number of rounds")
	cmd.Flags().IntVar(&iterations, "iterations", defaultCfg.Iterations, "Number of iterations")
	return cmd
}
