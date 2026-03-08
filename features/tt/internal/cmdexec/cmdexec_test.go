package cmdexec_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/cmdexec"
	"github.com/axsh/tokotachi/features/tt/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRunner(verbose bool) (*cmdexec.Runner, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := log.New(&buf, verbose)
	rec := cmdexec.NewRecorder()
	return &cmdexec.Runner{Logger: logger, DryRun: false, Recorder: rec}, &buf
}

func TestRun_Success(t *testing.T) {
	r, _ := newTestRunner(true)
	out, err := r.Run("echo", "hello")
	require.NoError(t, err)
	assert.Contains(t, out, "hello")
	recs := r.Recorder.Records()
	require.Len(t, recs, 1)
	assert.True(t, recs[0].Success)
	assert.Equal(t, 0, recs[0].ExitCode)
	assert.Contains(t, recs[0].Command, "echo hello")
}

func TestRun_Failure(t *testing.T) {
	r, _ := newTestRunner(true)
	_, err := r.Run("false")
	require.Error(t, err)
	recs := r.Recorder.Records()
	require.Len(t, recs, 1)
	assert.False(t, recs[0].Success)
}

func TestRun_DryRun(t *testing.T) {
	r, buf := newTestRunner(true)
	r.DryRun = true
	out, err := r.Run("echo", "should-not-run")
	require.NoError(t, err)
	assert.Empty(t, out)
	assert.Contains(t, buf.String(), "[DRY-RUN]")
	recs := r.Recorder.Records()
	require.Len(t, recs, 1)
	assert.True(t, recs[0].DryRun)
}

func TestLogPrefix_Normal(t *testing.T) {
	r, buf := newTestRunner(true)
	_, _ = r.Run("echo", "test")
	assert.Contains(t, buf.String(), "[CMD]")
}

func TestLogPrefix_DryRun(t *testing.T) {
	r, buf := newTestRunner(true)
	r.DryRun = true
	_, _ = r.Run("echo", "test")
	assert.Contains(t, buf.String(), "[DRY-RUN]")
	assert.NotContains(t, buf.String(), "[CMD] echo")
}

func TestRecorder_Collect(t *testing.T) {
	r, _ := newTestRunner(false)
	_, _ = r.Run("echo", "one")
	_, _ = r.Run("echo", "two")
	_, _ = r.Run("echo", "three")
	recs := r.Recorder.Records()
	assert.Len(t, recs, 3)
}

func TestResolveCommand_Default(t *testing.T) {
	got := cmdexec.ResolveCommand("TT_TEST_NONEXISTENT_XYZ_001", "fallback-cmd")
	assert.Equal(t, "fallback-cmd", got)
}

func TestResolveCommand_EnvOverride(t *testing.T) {
	t.Setenv("TT_TEST_CMD_RESOLVE_002", "/custom/path/mycmd")
	got := cmdexec.ResolveCommand("TT_TEST_CMD_RESOLVE_002", "default-cmd")
	assert.Equal(t, "/custom/path/mycmd", got)
}

func TestRunWithOpts_FailLevelDebug(t *testing.T) {
	r, buf := newTestRunner(true)
	_, err := r.RunWithOpts(cmdexec.CheckOpt(), "false")
	require.Error(t, err)
	logOut := buf.String()
	assert.NotContains(t, logOut, "[ERROR]", "CheckOpt should not produce ERROR logs")
	assert.Contains(t, logOut, "[DEBUG]", "CheckOpt should produce DEBUG logs")
	assert.Contains(t, logOut, "[SKIP]", "CheckOpt should use SKIP label")
}

func TestRunWithOpts_DefaultIsError(t *testing.T) {
	r, buf := newTestRunner(true)
	_, err := r.RunWithOpts(cmdexec.RunOption{}, "false")
	require.Error(t, err)
	logOut := buf.String()
	assert.Contains(t, logOut, "[ERROR]", "Default RunOption should produce ERROR logs")
	assert.Contains(t, logOut, "[FAIL]", "Default RunOption should use FAIL label")
}

func TestRunWithOpts_CustomLabel(t *testing.T) {
	r, buf := newTestRunner(true)
	_, err := r.RunWithOpts(cmdexec.RunOption{
		FailLevel:    log.LevelWarn,
		FailLevelSet: true,
		FailLabel:    "TOLERATED",
	}, "false")
	require.Error(t, err)
	logOut := buf.String()
	assert.Contains(t, logOut, "[WARN]", "Custom level should produce WARN logs")
	assert.Contains(t, logOut, "[TOLERATED]", "Custom label should appear in logs")
	assert.NotContains(t, logOut, "[ERROR]", "Should not produce ERROR logs")
}

func TestRunWithOpts_QuietCmd(t *testing.T) {
	// When QuietCmd=true and verbose=false, [CMD] should not appear
	rQuiet, bufQuiet := newTestRunner(false)
	_, _ = rQuiet.RunWithOpts(cmdexec.RunOption{QuietCmd: true}, "echo", "hello")
	assert.NotContains(t, bufQuiet.String(), "[CMD]", "QuietCmd with verbose=false should hide [CMD]")

	// When QuietCmd=false and verbose=false, [CMD] should appear as [INFO]
	rLoud, bufLoud := newTestRunner(false)
	_, _ = rLoud.RunWithOpts(cmdexec.RunOption{QuietCmd: false}, "echo", "hello")
	assert.Contains(t, bufLoud.String(), "[CMD]", "QuietCmd=false should show [CMD]")
}

func TestRunWithOpts_Dir(t *testing.T) {
	tmpDir := t.TempDir()
	r, _ := newTestRunner(true)

	// Run pwd/cd in the specified directory to verify cwd
	var out string
	var err error
	if runtime.GOOS == "windows" {
		out, err = r.RunWithOpts(cmdexec.RunOption{Dir: tmpDir}, "cmd", "/C", "cd")
	} else {
		out, err = r.RunWithOpts(cmdexec.RunOption{Dir: tmpDir}, "pwd")
	}
	require.NoError(t, err)

	// Resolve symlinks for comparison (TempDir may use symlinks)
	resolvedTmp, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	resolvedOut, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	require.NoError(t, err)

	assert.Equal(t, filepath.Clean(resolvedTmp), filepath.Clean(resolvedOut),
		"Command should run in the specified directory")
}

func TestRunWithOpts_DirEmpty(t *testing.T) {
	r, _ := newTestRunner(true)

	// When Dir is empty, command runs in the current process directory
	cwd, err := os.Getwd()
	require.NoError(t, err)

	var out string
	if runtime.GOOS == "windows" {
		out, err = r.RunWithOpts(cmdexec.RunOption{}, "cmd", "/C", "cd")
	} else {
		out, err = r.RunWithOpts(cmdexec.RunOption{}, "pwd")
	}
	require.NoError(t, err)

	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	require.NoError(t, err)
	resolvedOut, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	require.NoError(t, err)

	assert.Equal(t, filepath.Clean(resolvedCwd), filepath.Clean(resolvedOut),
		"Without Dir, command should run in process cwd")
}

func TestRunInteractiveWithOpts_Dir(t *testing.T) {
	tmpDir := t.TempDir()
	r, _ := newTestRunner(true)

	// RunInteractive attaches stdin/stdout/stderr, so we verify no error
	// using a simple command that exits immediately
	var err error
	if runtime.GOOS == "windows" {
		err = r.RunInteractiveWithOpts(cmdexec.RunOption{Dir: tmpDir}, "cmd", "/C", "echo ok")
	} else {
		err = r.RunInteractiveWithOpts(cmdexec.RunOption{Dir: tmpDir}, "true")
	}
	require.NoError(t, err)
}

func TestRun_DryRunWithDir(t *testing.T) {
	r, buf := newTestRunner(true)
	r.DryRun = true
	tmpDir := t.TempDir()

	_, err := r.RunWithOpts(cmdexec.RunOption{Dir: tmpDir}, "echo", "hello")
	require.NoError(t, err)
	logOut := buf.String()
	assert.Contains(t, logOut, "[DRY-RUN]", "DryRun log should appear")
	assert.Contains(t, logOut, tmpDir, "DryRun log should contain the Dir path")
}
