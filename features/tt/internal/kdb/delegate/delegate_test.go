package delegate

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireBash(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
}

func TestRun_Success(t *testing.T) {
	requireBash(t)
	out, err := Run(context.Background(), t.TempDir(), "echo hi")
	require.NoError(t, err)
	assert.Equal(t, "hi", strings.TrimSpace(out))
}

func TestRun_Failure(t *testing.T) {
	requireBash(t)
	_, err := Run(context.Background(), t.TempDir(), "false")
	assert.Error(t, err)
}

func TestRun_StderrInError(t *testing.T) {
	requireBash(t)
	_, err := Run(context.Background(), t.TempDir(), "echo boom >&2; exit 3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestRun_UsesWorkspaceDir(t *testing.T) {
	requireBash(t)
	dir := t.TempDir()
	out, err := Run(context.Background(), dir, "pwd")
	require.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(out))
}
