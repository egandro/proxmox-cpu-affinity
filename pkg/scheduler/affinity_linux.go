//go:build linux

package scheduler

import "golang.org/x/sys/unix"

// CPUSet is an alias for the Linux-specific CPU affinity mask.
type CPUSet = unix.CPUSet

// schedSetaffinity wraps the Linux sched_setaffinity syscall.
func schedSetaffinity(pid int, mask *CPUSet) error {
	return unix.SchedSetaffinity(pid, mask)
}
