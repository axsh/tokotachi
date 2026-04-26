package action

import (
	"strings"
	"testing"

	"github.com/axsh/tokotachi/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestDecidePendingChangesWithInput_YesFlag_WithPending_ForcesDelete(t *testing.T) {
	logger := log.New(testDiscardWriter{}, false)
	changes := PendingChanges{
		UntrackedFiles: []string{"tmp.txt"},
	}

	decision := decidePendingChangesWithInput(logger, changes, true, false, nil, "Proceed? [y/N]: ")
	assert.True(t, decision.Approved)
	assert.True(t, decision.ForceDelete)
}

func TestDecidePendingChangesWithInput_ConfirmYes_WithPending_ForcesDelete(t *testing.T) {
	logger := log.New(testDiscardWriter{}, false)
	changes := PendingChanges{
		UntrackedFiles: []string{"tmp.txt"},
	}

	decision := decidePendingChangesWithInput(
		logger,
		changes,
		false,
		false,
		strings.NewReader("yes\n"),
		"Proceed? [y/N]: ",
	)
	assert.True(t, decision.Approved)
	assert.True(t, decision.ForceDelete)
}

func TestDecidePendingChangesWithInput_ConfirmNo_Aborts(t *testing.T) {
	logger := log.New(testDiscardWriter{}, false)
	changes := PendingChanges{
		UntrackedFiles: []string{"tmp.txt"},
	}

	decision := decidePendingChangesWithInput(
		logger,
		changes,
		false,
		false,
		strings.NewReader("n\n"),
		"Proceed? [y/N]: ",
	)
	assert.False(t, decision.Approved)
	assert.False(t, decision.ForceDelete)
}

type testDiscardWriter struct{}

func (testDiscardWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
