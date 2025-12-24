package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/egandro/proxmox-cpu-affinity/pkg/hook"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <vmid> <phase>\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "\nParameters:")
		fmt.Fprintln(os.Stderr, "  <vmid>   The ID of the Virtual Machine (e.g., 100)")
		fmt.Fprintln(os.Stderr, "  <phase>  The lifecycle phase of the VM")
		fmt.Fprintln(os.Stderr, "\nPhases:")
		fmt.Fprintln(os.Stderr, "  pre-start   Executed before the guest is started. Non-zero exit aborts start.")
		fmt.Fprintln(os.Stderr, "  post-start  Executed after the guest successfully started.")
		fmt.Fprintln(os.Stderr, "  pre-stop    Executed before stopping the guest via the API.")
		fmt.Fprintln(os.Stderr, "  post-stop   Executed after the guest stopped.")
		flag.PrintDefaults()
	}

	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	vmidStr := args[0]
	vmid, err := strconv.Atoi(vmidStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: vmid must be an integer, got '%s'\n", vmidStr)
		os.Exit(1)
	}
	phase := args[1]

	// fmt.Printf("GUEST HOOK: %s\n", strings.Join(args, " "))

	h := hook.New()
	if err := h.Handle(vmid, phase); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
