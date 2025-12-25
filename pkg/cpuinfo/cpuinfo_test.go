package cpuinfo

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCoreRanking_OneIteration(t *testing.T) {
	c := New()
	// Run 1 round with 1 iteration to verify the loop logic without heavy load
	err := c.Update(1, 1, nil)
	if err != nil {
		t.Logf("Update failed (expected in non-Linux/restricted envs): %v", err)
		return
	}

	rankings, err := c.GetCoreRanking()

	if err != nil {
		t.Logf("GetCoreRanking failed (expected in non-Linux/restricted envs): %v", err)
		return
	}

	assert.NotEmpty(t, rankings, "Expected rankings to be non-empty on success")
}

func TestCalculateRanking_Success(t *testing.T) {
	c := New()
	// Run with minimal work and generous timeout
	err := c.CalculateRanking(1, 1, 10*time.Second)

	// On supported platforms, this should be nil.
	// On unsupported platforms, it returns an error, but NOT a timeout.
	if err != nil {
		if strings.Contains(err.Error(), "timed out") {
			t.Errorf("CalculateRanking timed out unexpectedly: %v", err)
		} else {
			t.Logf("CalculateRanking returned platform error: %v", err)
		}
	} else {
		// If success, verify we have rankings
		rankings, err := c.GetCoreRanking()
		assert.NoError(t, err)
		assert.NotEmpty(t, rankings)
	}
}

func TestCalculateRanking_Timeout(t *testing.T) {
	c := New()
	// Run with work that definitely takes > 1ns
	// Note: On non-Linux, Update returns error immediately, so we might not hit timeout.
	err := c.CalculateRanking(10, 1000, 1*time.Nanosecond)

	assert.Error(t, err, "Expected an error (timeout or platform specific)")

	if err != nil && strings.Contains(err.Error(), "timed out") {
		// This is the success path for the timeout test on Linux
		t.Log("Successfully caught timeout")
	}
}

func TestSelectCores_EdgeCases(t *testing.T) {
	c := New()
	// Initialize with dummy data if possible, or run a quick update
	// Since we can't easily inject cache without Update, we run a quick update
	if err := c.Update(1, 1, nil); err != nil {
		t.Skipf("Skipping affinity test due to update failure (non-Linux?): %v", err)
	}

	// Request 1 core
	cpus, err := c.SelectCores(1)
	assert.NoError(t, err)
	assert.Len(t, cpus, 1)

	// Request too many
	_, err = c.SelectCores(9999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed available")

	// Request 0 cores
	_, err = c.SelectCores(0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "greater than 0")
}

func TestSelectCores_Race(t *testing.T) {
	c := New()
	// Initial update to ensure we have data
	if err := c.Update(1, 1, nil); err != nil {
		t.Skipf("Skipping race test due to update failure: %v", err)
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writer goroutine (Updates topology)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				// Run a quick update
				_ = c.Update(1, 1, nil)
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	// Reader goroutines (SelectCores)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				cpus, err := c.SelectCores(1)
				// It's possible Update fails on some platforms or transiently,
				// but SelectCores should generally succeed if cache is populated.
				// We mainly care that it doesn't panic or race.
				if err == nil {
					assert.NotEmpty(t, cpus)
				}
			}
		}()
	}

	// Let them race for a bit
	time.Sleep(100 * time.Millisecond)
	close(done)
	wg.Wait()
}
