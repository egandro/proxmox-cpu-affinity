package main

import (
	"context"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/executor"
)

func FuzzCheckStorage(f *testing.F) {
	f.Add([]byte("local dir active\n"))
	f.Add([]byte("local dir disabled\n"))
	f.Add([]byte("other zfs active\n"))
	f.Add([]byte(""))
	f.Add([]byte("invalid output"))

	f.Fuzz(func(t *testing.T, pvesmOutput []byte) {
		mockExec := &executor.MockExecutor{
			OutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return pvesmOutput, nil
			},
		}
		_ = checkStorage(context.Background(), mockExec, "local")
	})
}

func FuzzGetAllVMIDs(f *testing.F) {
	f.Add([]byte("100 running\n101 stopped\n"))
	f.Add([]byte("   100   \n"))
	f.Add([]byte("not a number\n"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, qmListOutput []byte) {
		mockExec := &executor.MockExecutor{
			OutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return qmListOutput, nil
			},
		}
		_ = getAllVMIDs(context.Background(), mockExec)
	})
}
