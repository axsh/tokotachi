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
	// Create .git file to simulate valid worktree
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../../.git/worktrees/test-001\n"), 0o644))
	assert.True(t, m.Exists("test-001"))
}

func TestExists_False(t *testing.T) {
	m := newTestManager(t, true)
	assert.False(t, m.Exists("nonexistent"))
}

func TestExists_GhostDirectory(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("ghost-branch")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// No .git file — ghost directory
	assert.False(t, m.Exists("ghost-branch"))
}

func TestExists_ValidWorktree(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("valid-branch")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../../.git/worktrees/valid-branch\n"), 0o644))
	assert.True(t, m.Exists("valid-branch"))
}

func TestCreate_CleansGhostDirectory(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("ghost-branch")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// No .git file = ghost directory
	err := m.Create("ghost-branch")
	require.NoError(t, err)
	// Dry-run mode: git worktree add command should be recorded
	recs := m.CmdRunner.Recorder.Records()
	require.GreaterOrEqual(t, len(recs), 1)
}

func TestRemove_CleansRemainingDirectory(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("leftover-branch")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	err := m.Remove("leftover-branch", false)
	require.NoError(t, err)
	// Directory should be removed after Remove() post-cleanup
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr), "directory should be removed after Remove()")
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

func TestFindNestedWorktrees_WithChildren(t *testing.T) {
	m := newTestManager(t, true)

	// Create parent worktree directory
	parentDir := m.Path("parent")
	require.NoError(t, os.MkdirAll(parentDir, 0o755))

	// Create child-a with .git file (valid worktree)
	childA := filepath.Join(parentDir, "work", "child-a")
	require.NoError(t, os.MkdirAll(childA, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(childA, ".git"),
		[]byte("gitdir: ../../../../.git/worktrees/child-a\n"), 0o644,
	))

	// Create child-b without .git file (ghost directory — should be excluded)
	childB := filepath.Join(parentDir, "work", "child-b")
	require.NoError(t, os.MkdirAll(childB, 0o755))

	result := m.FindNestedWorktrees("parent")
	assert.Equal(t, []string{"child-a"}, result)
}

func TestFindNestedWorktrees_NoChildren(t *testing.T) {
	m := newTestManager(t, true)

	// Create parent worktree directory without work/ subdirectory
	parentDir := m.Path("parent")
	require.NoError(t, os.MkdirAll(parentDir, 0o755))

	result := m.FindNestedWorktrees("parent")
	assert.Empty(t, result)
}

func TestFindNestedWorktrees_NoWorkDir(t *testing.T) {
	m := newTestManager(t, true)

	// parent directory does not exist at all
	result := m.FindNestedWorktrees("nonexistent")
	assert.Empty(t, result)
}

func TestPrune(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Prune()
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.Len(t, recs, 1)
	assert.Contains(t, recs[0].Command, "worktree prune")
}
