package main

import (
	"context"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/executor"
)

func FuzzCheckStorage(f *testing.F) {
	f.Add([]byte("proxmox-backup-client failed: Error: unable to open chunk store at \"/pbs/.chunks\" - No such file or directory (os error 2)"))
	f.Add([]byte("Name         Type     Status     Total (KiB)      Used (KiB) Available (KiB)        %"))
	f.Add([]byte("local         dir     active               0               0               0      N/A"))
	f.Add([]byte("invalid output"))
	f.Add([]byte("pbs           pbs   inactive               0               0               0    0.00%"))
	f.Add([]byte("old           dir     active         1921598         1112718            7484   57.91%"))

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
	f.Add([]byte("ct: 100"))
	f.Add([]byte("    group HA"))
	f.Add([]byte("    state started"))
	f.Add([]byte("invalid output\n"))
	f.Add([]byte("vm: 103"))

	f.Fuzz(func(t *testing.T, qmListOutput []byte) {
		mockExec := &executor.MockExecutor{
			OutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return qmListOutput, nil
			},
		}
		_ = getVMIDsFromQMList(context.Background(), mockExec)
	})
}

func FuzzIsHAVM(f *testing.F) {
	f.Add([]byte("ct: 100"))
	f.Add([]byte("    group HA"))
	f.Add([]byte("    state started"))
	f.Add([]byte("invalid output\n"))
	f.Add([]byte("vm: 103"))

	f.Fuzz(func(t *testing.T, haOutput []byte) {
		// Reset global cache for each iteration to ensure parsing logic is triggered
		haConfigMu.Lock()
		haConfigLoaded = false
		haConfigCache = ""
		haConfigMu.Unlock()

		mockExec := &executor.MockExecutor{
			OutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return haOutput, nil
			},
		}
		_ = isHAVM(context.Background(), mockExec, 100)
	})
}
