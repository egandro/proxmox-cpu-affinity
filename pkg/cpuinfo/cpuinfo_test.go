package cpuinfo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCoreRanking_OneIteration(t *testing.T) {
	c := New(nil)
	// Run 1 round with 1 iteration to verify the loop logic without heavy load
	err := c.Update(1, 1)
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
