package main

import (
	"context"
	"testing"

	"github.com/egandro/proxmox-cpu-affinity/pkg/executor"
)

func FuzzGetVMProcessInfo_QMConfig(f *testing.F) {
	// Seed with valid JSON
	f.Add([]byte(`{"cores": 4, "sockets": 1}`))
	f.Add([]byte(`{"cores": 0, "sockets": 0}`))
	f.Add([]byte(`{"cores": 4, "sockets": 1, "affinity": "0-3"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"hookscript": "local:snippets/hook.pl"}`))

	// Seed with edge cases
	f.Add([]byte(`{"cores": -1}`))
	f.Add([]byte(`{"cores": 9999999999}`))
	f.Add([]byte(`not valid json`))
	f.Add([]byte(`{"cores": "string-not-int"}`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, qmOutput []byte) {
		mockExec := &executor.MockExecutor{
			OutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
				return qmOutput, nil
			},
		}

		mockPIDReader := func(vmid uint64) ([]byte, error) {
			return []byte("12345"), nil
		}

		_ = getVMProcessInfo(context.Background(), mockExec, mockPIDReader, 100, true, false)
	})
}
