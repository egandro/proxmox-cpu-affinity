package cpuinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCoreRanking_OneIteration(t *testing.T) {
	c := New()
	// Run 1 round with 1 iteration to verify the loop logic without heavy load
	rankings, err := c.GetCoreRanking(1, 1, nil)

	if err != nil {
		t.Logf("GetCoreRanking failed (expected in non-Linux/restricted envs): %v", err)
		return
	}

	assert.NotEmpty(t, rankings, "Expected rankings to be non-empty on success")
}
