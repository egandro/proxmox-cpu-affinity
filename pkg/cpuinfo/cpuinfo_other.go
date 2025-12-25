//go:build !linux

package cpuinfo

import "errors"

// lockToCPU is not supported on non-Linux platforms.
// CPU affinity measurement requires Linux-specific syscalls.
func lockToCPU(cpuID int) error {
	return errors.New("CPU affinity is only supported on Linux")
}
