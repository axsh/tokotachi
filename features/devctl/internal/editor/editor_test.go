package editor_test

import (
	"bytes"
	"testing"

	"github.com/escape-dev/devctl/internal/cmdexec"
	"github.com/escape-dev/devctl/internal/detect"
	"github.com/escape-dev/devctl/internal/editor"
	"github.com/escape-dev/devctl/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLauncher(t *testing.T) {
	tests := []struct {
		ed   detect.Editor
		name string
	}{
		{detect.EditorVSCode, "code"},
		{detect.EditorCursor, "cursor"},
		{detect.EditorAG, "ag"},
		{detect.EditorClaude, "claude"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := editor.NewLauncher(tt.ed)
			require.NoError(t, err)
			assert.Equal(t, tt.name, l.Name())
		})
	}
}

func TestNewLauncher_Invalid(t *testing.T) {
	_, err := editor.NewLauncher(detect.Editor("vim"))
	require.Error(t, err)
}

func TestLauncher_DryRun(t *testing.T) {
	editors := []detect.Editor{
		detect.EditorVSCode, detect.EditorCursor,
		detect.EditorAG, detect.EditorClaude,
	}
	for _, ed := range editors {
		t.Run(string(ed), func(t *testing.T) {
			var buf bytes.Buffer
			logger := log.New(&buf, true)
			rec := cmdexec.NewRecorder()
			runner := &cmdexec.Runner{Logger: logger, DryRun: true, Recorder: rec}
			l, err := editor.NewLauncher(ed)
			require.NoError(t, err)

			result, err := l.Launch(editor.LaunchOptions{
				WorktreePath: "/tmp/test-worktree",
				DryRun:       true,
				Logger:       logger,
				CmdRunner:    runner,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, result.Method)
			assert.Contains(t, buf.String(), "[DRY-RUN]")
		})
	}
}

func TestResolveCommand_Default(t *testing.T) {
	got := editor.ResolveCommand("DEVCTL_TEST_NONEXISTENT_VAR_12345", "fallback-cmd")
	assert.Equal(t, "fallback-cmd", got)
}

func TestResolveCommand_EnvOverride(t *testing.T) {
	t.Setenv("DEVCTL_TEST_CMD_OVERRIDE", "/custom/path/myeditor")
	got := editor.ResolveCommand("DEVCTL_TEST_CMD_OVERRIDE", "default-cmd")
	assert.Equal(t, "/custom/path/myeditor", got)
}
