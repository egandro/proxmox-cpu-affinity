//go:build linux

package cpuinfo

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/unix"
)

func lockToCPU(cpuID int) error {
	runtime.LockOSThread()
	var mask unix.CPUSet
	mask.Set(cpuID)
	if err := unix.SchedSetaffinity(0, &mask); err != nil {
		return fmt.Errorf("failed to lock thread to CPU %d: %w", cpuID, err)
	}
	return nil
}
