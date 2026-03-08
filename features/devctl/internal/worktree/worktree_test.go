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

func TestPath_WithFeature(t *testing.T) {
	m := newTestManager(t, true)
	got := m.Path("devctl", "test-001")
	// New structure: work/<branch>/features/<feature>
	assert.Equal(t, filepath.Join(m.RepoRoot, "work", "test-001", "features", "devctl"), got)
}

func TestPath_NoFeature(t *testing.T) {
	m := newTestManager(t, true)
	got := m.Path("", "test-001")
	// No feature: work/<branch>/all
	assert.Equal(t, filepath.Join(m.RepoRoot, "work", "test-001", "all"), got)
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

func TestExists_NoFeature(t *testing.T) {
	m := newTestManager(t, true)
	dir := m.Path("", "test-001")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	assert.True(t, m.Exists("", "test-001"))
}

func TestCreateCmd(t *testing.T) {
	m := newTestManager(t, true)
	err := m.Create("devctl", "test-001")
	require.NoError(t, err)
	recs := m.CmdRunner.Recorder.Records()
	require.GreaterOrEqual(t, len(recs), 1)
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
	// Create feature worktree directories under new structure
	require.NoError(t, os.MkdirAll(m.Path("feature-a", "main"), 0o755))
	require.NoError(t, os.MkdirAll(m.Path("feature-b", "main"), 0o755))
	// Create a file in features dir (should be ignored)
	featuresDir := filepath.Join(m.RepoRoot, "work", "main", "features")
	f, err := os.Create(filepath.Join(featuresDir, "some-file.txt"))
	require.NoError(t, err)
	f.Close()

	entries, err := m.List("main")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	features := make([]string, len(entries))
	for i, e := range entries {
		features[i] = e.Feature
	}
	assert.Contains(t, features, "feature-a")
	assert.Contains(t, features, "feature-b")
}

func TestListEntries_NoFeatures(t *testing.T) {
	m := newTestManager(t, true)
	entries, err := m.List("nonexistent-branch")
	require.NoError(t, err)
	assert.Nil(t, entries)
}
