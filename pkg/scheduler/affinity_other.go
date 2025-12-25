//go:build !linux

package scheduler

import "errors"

// CPUSet is a stub type for non-Linux platforms.
// The actual implementation only works on Linux.
type CPUSet struct {
	bits [16]uint64 // matches unix.CPUSet size
}

// Set marks cpu as part of the set.
func (s *CPUSet) Set(cpu int) {
	if cpu >= 0 && cpu < 1024 {
		s.bits[cpu/64] |= 1 << (uint(cpu) % 64)
	}
}

// IsSet reports whether cpu is part of the set.
func (s *CPUSet) IsSet(cpu int) bool {
	if cpu >= 0 && cpu < 1024 {
		return s.bits[cpu/64]&(1<<(uint(cpu)%64)) != 0
	}
	return false
}

// schedSetaffinity is a stub that returns an error on non-Linux platforms.
func schedSetaffinity(pid int, mask *CPUSet) error {
	return errors.New("CPU affinity is only supported on Linux")
}
