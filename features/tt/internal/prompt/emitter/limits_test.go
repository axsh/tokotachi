package emitter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLimits(t *testing.T) {
	target := &manifest.Entity{
		ID:   "antigravity",
		Kind: "target",
		Raw: map[string]any{
			"limits": map[string]any{
				"rules": map[string]any{
					"max_file_size": 100,
					"on_exceed":     "warn",
				},
			},
		},
	}

	tl := ExtractLimits(target)
	require.NotNil(t, tl)
	require.NotNil(t, tl.Rules)
	assert.Equal(t, 100, tl.Rules.MaxFileSize)
	assert.Equal(t, "warn", tl.Rules.OnExceed)
	assert.Nil(t, tl.Skills)
	assert.Nil(t, tl.Workflows)
}

func TestExtractLimits_NoLimits(t *testing.T) {
	target := &manifest.Entity{
		ID:   "antigravity",
		Kind: "target",
		Raw:  map[string]any{},
	}

	tl := ExtractLimits(target)
	assert.Nil(t, tl)
}

func TestExtractLimits_NilTarget(t *testing.T) {
	tl := ExtractLimits(nil)
	assert.Nil(t, tl)
}

func TestCheckAndApplyLimit_NoLimit(t *testing.T) {
	content := "short content"
	result, shouldWrite, err := CheckAndApplyLimit(content, nil, "test", ".")
	require.NoError(t, err)
	assert.True(t, shouldWrite)
	assert.Equal(t, content, result)
}

func TestCheckAndApplyLimit_WithinLimit(t *testing.T) {
	content := "short"
	limit := &CategoryLimit{MaxFileSize: 100, OnExceed: "error"}
	result, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", ".")
	require.NoError(t, err)
	assert.True(t, shouldWrite)
	assert.Equal(t, content, result)
}

func TestCheckAndApplyLimit_Error(t *testing.T) {
	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{MaxFileSize: 100, OnExceed: "error"}
	_, shouldWrite, err := CheckAndApplyLimit(content, limit, "test-entity", ".")
	require.Error(t, err)
	assert.False(t, shouldWrite)
	assert.Contains(t, err.Error(), "exceeds limit")
	assert.Contains(t, err.Error(), "test-entity")
}

func TestCheckAndApplyLimit_Warn(t *testing.T) {
	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{MaxFileSize: 100, OnExceed: "warn"}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	result, shouldWrite, err := CheckAndApplyLimit(content, limit, "test-entity", ".")

	w.Close()
	os.Stderr = oldStderr
	stderrOutput := make([]byte, 1024)
	n, _ := r.Read(stderrOutput)
	r.Close()

	require.NoError(t, err)
	assert.True(t, shouldWrite)
	assert.Equal(t, content, result)
	assert.Contains(t, string(stderrOutput[:n]), "WARNING")
}

func TestCheckAndApplyLimit_Truncate(t *testing.T) {
	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{MaxFileSize: 100, OnExceed: "truncate"}
	result, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", ".")
	require.NoError(t, err)
	assert.True(t, shouldWrite)
	assert.Len(t, result, 100)
}

func TestCheckAndApplyLimit_Skip(t *testing.T) {
	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{MaxFileSize: 100, OnExceed: "skip"}
	result, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", ".")
	require.NoError(t, err)
	assert.False(t, shouldWrite)
	assert.Equal(t, "", result)
}

func TestCheckAndApplyLimit_BashScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test script that outputs "SUMMARIZED"
	scriptName := "summarize.sh"
	scriptPath := filepath.Join(tmpDir, scriptName)
	scriptContent := "#!/bin/bash\necho -n \"SUMMARIZED\"\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{
		MaxFileSize: 100,
		OnExceed:    "bash:" + scriptName, // relative to rootDir
	}

	result, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", tmpDir)
	require.NoError(t, err)
	assert.True(t, shouldWrite)
	assert.Equal(t, "SUMMARIZED", result)
}

func TestCheckAndApplyLimit_BashScript_ReceivesPromptArg(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that reads the file passed via --prompt and echoes its content
	scriptName := "echo_prompt.sh"
	scriptPath := filepath.Join(tmpDir, scriptName)
	scriptContent := `#!/bin/bash
while [[ $# -gt 0 ]]; do
  case "$1" in
    --prompt) PROMPT_FILE="$2"; shift 2 ;;
    *) shift ;;
  esac
done
cat "$PROMPT_FILE"
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	content := "original content to process"
	limit := &CategoryLimit{
		MaxFileSize: 10,
		OnExceed:    "bash:" + scriptName,
	}

	result, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", tmpDir)
	require.NoError(t, err)
	assert.True(t, shouldWrite)
	assert.Equal(t, content, result)
}

func TestCheckAndApplyLimit_BashScript_NotFound(t *testing.T) {
	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{
		MaxFileSize: 100,
		OnExceed:    "bash:nonexistent/script.sh",
	}

	_, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", t.TempDir())
	require.Error(t, err)
	assert.False(t, shouldWrite)
	assert.Contains(t, err.Error(), "not found")
}

func TestCheckAndApplyLimit_BashScript_Error(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that exits with error
	scriptName := "fail.sh"
	scriptPath := filepath.Join(tmpDir, scriptName)
	scriptContent := "#!/bin/bash\nexit 1\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))

	content := strings.Repeat("x", 200)
	limit := &CategoryLimit{
		MaxFileSize: 100,
		OnExceed:    "bash:" + scriptName,
	}

	_, shouldWrite, err := CheckAndApplyLimit(content, limit, "test", tmpDir)
	require.Error(t, err)
	assert.False(t, shouldWrite)
	assert.Contains(t, err.Error(), "script failed")
}
