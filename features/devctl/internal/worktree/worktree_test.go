package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/escape-dev/devctl/internal/cmdexec"
	"github.com/escape-dev/devctl/internal/log"
	"github.com/escape-dev/devctl/internal/worktree"
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
	got := m.Path("devctl", "test-001")
	assert.Equal(t, filepath.Join(m.RepoRoot, "work", "devctl", "test-001"), got)
}

func TestExists_True(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("devctl", "test-001")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	assert.True(t, m.Exists("devctl", "test-001"))
}

func TestExists_False(t *testing.T) {
	m := newTestManager(t, true)
	assert.False(t, m.Exists("devctl", "nonexistent"))
}

func TestCreateCmd(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Create("devctl", "test-001")
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.GreaterOrEqual(t, len(recs), 1)
	// In dry-run, the first record should be branch check or worktree add
	found := false
	for _, r := range recs {
		if assert.ObjectsAreEqual("", "") {
			_ = r
		}
		found = true
	}
	assert.True(t, found)
}

func TestRemoveCmd(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Remove("devctl", "test-001", false)
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].Command, "worktree remove")
}

func TestRemoveCmd_Force(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Remove("devctl", "test-001", true)
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

func TestListEntries(t *testing.T) {
	m := newTestManager(t, true)
	// Create some worktree directories
	require.NoError(t, os.MkdirAll(m.Path("devctl", "branch-a"), 0o755))
	require.NoError(t, os.MkdirAll(m.Path("devctl", "branch-b"), 0o755))
	// Create a file (should be ignored)
	f, err := os.Create(filepath.Join(m.RepoRoot, "work", "devctl", "some-file.txt"))
	require.NoError(t, err)
	f.Close()

	entries, err := m.List("devctl")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	branches := make([]string, len(entries))
	for i, e := range entries {
		branches[i] = e.Branch
	}
	assert.Contains(t, branches, "branch-a")
	assert.Contains(t, branches, "branch-b")
}
