package forge

import (
	"io"
	"testing"
	"time"

	"github.com/axsh/tokotachi/internal/cmdexec"
	"github.com/axsh/tokotachi/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRunner(t *testing.T) (*cmdexec.Runner, *cmdexec.Recorder) {
	t.Helper()
	rec := cmdexec.NewRecorder()
	logger := log.New(io.Discard, false)
	return &cmdexec.Runner{Logger: logger, DryRun: true, Recorder: rec}, rec
}

func TestNewClient_SelectsGitHub(t *testing.T) {
	runner, _ := newTestRunner(t)
	c, err := NewClient(Options{Host: HostGitHub, CmdRunner: runner})
	require.NoError(t, err)
	assert.Equal(t, HostGitHub, c.Host())
}

func TestNewClient_SelectsGitLab(t *testing.T) {
	runner, _ := newTestRunner(t)
	c, err := NewClient(Options{Host: HostGitLab, CmdRunner: runner})
	require.NoError(t, err)
	assert.Equal(t, HostGitLab, c.Host())
}

func TestNewClient_UnknownHost_DefaultsGitHub(t *testing.T) {
	runner, _ := newTestRunner(t)
	c, err := NewClient(Options{Host: HostUnknown, CmdRunner: runner})
	require.NoError(t, err)
	assert.Equal(t, HostGitHub, c.Host())
}

func TestNewClient_NilRunner_Error(t *testing.T) {
	_, err := NewClient(Options{Host: HostGitHub})
	require.Error(t, err)
}

func TestGitHubAdapter_CreateChangeRequest_InvokesGhPrCreate(t *testing.T) {
	runner, rec := newTestRunner(t)
	c, err := NewClient(Options{Host: HostGitHub, CmdRunner: runner, GhCmd: "gh"})
	require.NoError(t, err)

	err = c.CreateChangeRequest("/tmp/work")
	require.NoError(t, err)

	recs := rec.Records()
	require.NotEmpty(t, recs)
	assert.Contains(t, recs[0].Command, "gh pr create")
}

func TestGitHubAdapter_FindOpen_InvokesGhPrList(t *testing.T) {
	runner, rec := newTestRunner(t)
	c, err := NewClient(Options{Host: HostGitHub, CmdRunner: runner, GhCmd: "gh"})
	require.NoError(t, err)

	info, err := c.FindOpenChangeRequest("/tmp/work", "feat-x")
	require.NoError(t, err)
	assert.Nil(t, info) // DryRun returns empty output

	recs := rec.Records()
	require.NotEmpty(t, recs)
	assert.Contains(t, recs[0].Command, "gh pr list")
	assert.Contains(t, recs[0].Command, "--head feat-x")
}

func TestGitLabAdapter_CreateChangeRequest_InvokesGlabMrCreate(t *testing.T) {
	runner, rec := newTestRunner(t)
	c, err := NewClient(Options{Host: HostGitLab, CmdRunner: runner, GlabCmd: "glab"})
	require.NoError(t, err)

	err = c.CreateChangeRequest("/tmp/work")
	require.NoError(t, err)

	recs := rec.Records()
	require.NotEmpty(t, recs)
	assert.Contains(t, recs[0].Command, "glab mr create")
}

func TestGitLabAdapter_FindOpen_ParsesJSON(t *testing.T) {
	raw := `[{"iid":42,"created_at":"2026-01-02T03:04:05Z"}]`
	info, err := parseGitLabMRListJSON(raw)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 42, info.Number)
	assert.True(t, info.CreatedAt.UTC().Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)))
}

func TestGitHubAdapter_FindOpen_ParsesJSON(t *testing.T) {
	raw := `[{"number":7,"createdAt":"2026-03-04T05:06:07Z"}]`
	info, err := parseGitHubPRListJSON(raw)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 7, info.Number)
}

func TestParseGitLabMRListJSON_Empty(t *testing.T) {
	info, err := parseGitLabMRListJSON("[]")
	require.NoError(t, err)
	assert.Nil(t, info)
}
