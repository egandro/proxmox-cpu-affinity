package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/egandro/proxmox-cpu-affinity/pkg/hook"
)

func main() {
	if _, err := os.Stat("/usr/sbin/proxmox-cpu-affinity-service"); os.IsNotExist(err) {
		// In case the VM is migrated to a system where "proxmox-cpu-affinity" is not installed,
		// we bypass the hook to avoid delaying VM startup due to timeouts.
		fmt.Fprintln(os.Stderr, "Warning: /usr/sbin/proxmox-cpu-affinity-service not found.\nThis hook requires the proxmox-cpu-affinity package to be installed.")
		os.Exit(0)
	}

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

	h := hook.New()
	if err := h.Handle(vmid, phase); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
