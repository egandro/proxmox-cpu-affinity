package config

import (
	"testing"
)

func FuzzGetEnvInt(f *testing.F) {
	// Seed with some interesting values
	f.Add("42")
	f.Add("-1")
	f.Add("0")
	f.Add("")
	f.Add("not-a-number")
	f.Add("9999999999999999999999")
	f.Add("  123  ")
	f.Add("1.5")

	f.Fuzz(func(t *testing.T, input string) {
		key := "FUZZ_TEST_INT"
		t.Setenv(key, input)

		// Should never panic, always return fallback or parsed value
		result := getEnvInt(key, 42)
		if result < 0 && result != 42 {
			// Negative values are valid, just checking it doesn't crash
			_ = result
		}
	})
}

func FuzzGetEnvBool(f *testing.F) {
	// Seed with interesting values
	f.Add("true")
	f.Add("false")
	f.Add("TRUE")
	f.Add("FALSE")
	f.Add("1")
	f.Add("0")
	f.Add("yes")
	f.Add("no")
	f.Add("")
	f.Add("maybe")

	f.Fuzz(func(t *testing.T, input string) {
		key := "FUZZ_TEST_BOOL"
		t.Setenv(key, input)

		// Should never panic
		_ = getEnvBool(key, false)
	})
}
