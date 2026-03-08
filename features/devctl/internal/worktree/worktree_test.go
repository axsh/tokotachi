package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
	"github.com/axsh/tokotachi/features/devctl/internal/log"
	"github.com/axsh/tokotachi/features/devctl/internal/worktree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T, dryRun bool) *worktree.Manager {
	t.Helper()
	var buf []byte
	_ = buf
	logger := log.New(os.Stderr, true)
	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: dryRun, Recorder: rec}
	return &worktree.Manager{CmdRunner: runner, RepoRoot: t.TempDir()}
}

func TestPath(t *testing.T) {
	m := newTestManager(t, true)
	got := m.Path("test-001")
	// Unified structure: work/<branch>
	assert.Equal(t, filepath.Join(m.RepoRoot, "work", "test-001"), got)
}

func TestExists_True(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("test-001")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	assert.True(t, m.Exists("test-001"))
}

func TestExists_False(t *testing.T) {
	m := newTestManager(t, true)
	assert.False(t, m.Exists("nonexistent"))
}

func TestCreateCmd(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Create("test-001")
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.GreaterOrEqual(t, len(recs), 1)
}

func TestRemoveCmd(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Remove("test-001", false)
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].Command, "worktree remove")
}

func TestRemoveCmd_Force(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Remove("test-001", true)
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].Command, "-f")
}

func TestDeleteBranchCmd(t *testing.T) {
	m := newTestManager(t, true)
	err := m.DeleteBranch("test-001", false)
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].Command, "branch -d")
}

func TestDeleteBranchCmd_Force(t *testing.T) {
	m := newTestManager(t, true)
	err := m.DeleteBranch("test-001", true)
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].Command, "branch -D")
}
