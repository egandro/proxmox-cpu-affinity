package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/egandro/proxmox-cpu-affinity/pkg/config"
	"github.com/egandro/proxmox-cpu-affinity/pkg/cpuinfo"
	"github.com/spf13/cobra"
)

func newCPUInfoCmd() *cobra.Command {
	var verbose bool
	var summary bool
	var rounds int
	var iterations int
	var quiet bool

	// Load config to get defaults
	defaultCfg := config.Load(config.ConstantConfigFilename)

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

			var s *spinner.Spinner
			if !verbose && !quiet {
				s = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
				s.Suffix = fmt.Sprintf(" Analyzing CPU topology (Rounds: %d, Iterations: %d)...", rounds, iterations)
				s.Start()
			}

			ci := cpuinfo.New()
			err := ci.Update(rounds, iterations, onProgress)

			if s != nil {
				s.Stop()
			}

			if err != nil {
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
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Disable progress spinner")
	return cmd
}
