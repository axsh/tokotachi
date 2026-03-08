package scaffold

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSpinner_StartStop(t *testing.T) {
	// Verify spinner doesn't panic
	s := NewSpinner(nil) // nil writer = discard output
	s.Start("Loading...")
	time.Sleep(250 * time.Millisecond)
	s.Stop()
}

func TestSpinner_UpdateMessage(t *testing.T) {
	s := NewSpinner(nil)
	s.Start("Phase 1...")
	time.Sleep(150 * time.Millisecond)
	s.UpdateMessage("Phase 2...")
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	// Just verify no panics
	assert.True(t, true)
}
