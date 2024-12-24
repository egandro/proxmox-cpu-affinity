package proxmox

import (
	"testing"
)

func FuzzGetVmConfigJSON(f *testing.F) {
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

	f.Fuzz(func(t *testing.T, data []byte) {
		mock := &MockExecutor{
			OutputFunc: func(name string, arg ...string) ([]byte, error) {
				return data, nil
			},
		}

		p := &proxmox{executor: mock, nodeName: "test"}

		// Should never panic, may return error for invalid JSON
		_, _ = p.GetVmConfig(100)
	})
}

func FuzzGetVmPidParse(f *testing.F) {
	// Seed with interesting PID values
	f.Add([]byte("12345"))
	f.Add([]byte("1"))
	f.Add([]byte("0"))
	f.Add([]byte("-1"))
	f.Add([]byte(""))
	f.Add([]byte("not-a-pid"))
	f.Add([]byte("  12345  \n"))
	f.Add([]byte("99999999999999"))

	f.Fuzz(func(t *testing.T, data []byte) {
		mockSys := &MockSystemOps{
			ReadFileFunc: func(name string) ([]byte, error) {
				return data, nil
			},
			ProcessExistsFunc: func(pid int) bool {
				return true // Always say process exists
			},
		}

		p := &proxmox{sys: mockSys}

		// Should never panic
		_, _ = p.GetVmPid(100)
	})
}
