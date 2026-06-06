package emitter

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// CategoryLimit defines size limit settings for a category (rules/skills/workflows).
type CategoryLimit struct {
	MaxFileSize int    // max bytes; 0 means no limit
	OnExceed    string // "error" | "warn" | "truncate" | "skip" | "bash:<script_path>"
}

// TargetLimits holds per-category limits.
type TargetLimits struct {
	Rules     *CategoryLimit
	Skills    *CategoryLimit
	Workflows *CategoryLimit
}

// ExtractLimits reads limits from a target entity's raw map.
// Returns nil if the target is nil or has no limits defined.
func ExtractLimits(target *manifest.Entity) *TargetLimits {
	if target == nil {
		return nil
	}

	limitsRaw, ok := target.Raw["limits"].(map[string]any)
	if !ok {
		return nil
	}

	tl := &TargetLimits{}
	tl.Rules = extractCategoryLimit(limitsRaw, "rules")
	tl.Skills = extractCategoryLimit(limitsRaw, "skills")
	tl.Workflows = extractCategoryLimit(limitsRaw, "workflows")

	// Return nil if all categories are nil
	if tl.Rules == nil && tl.Skills == nil && tl.Workflows == nil {
		return nil
	}

	return tl
}

// extractCategoryLimit extracts a CategoryLimit from the limits map for a given category key.
func extractCategoryLimit(limitsRaw map[string]any, key string) *CategoryLimit {
	catRaw, ok := limitsRaw[key].(map[string]any)
	if !ok {
		return nil
	}

	cl := &CategoryLimit{}

	switch v := catRaw["max_file_size"].(type) {
	case int:
		cl.MaxFileSize = v
	case float64:
		cl.MaxFileSize = int(v)
	}

	if s, ok := catRaw["on_exceed"].(string); ok {
		cl.OnExceed = s
	}

	return cl
}

// CheckAndApplyLimit checks content against limit and returns
// (processedContent, shouldWrite, error).
//
// If limit is nil or MaxFileSize is 0, the content is returned unchanged.
// If the content size is within the limit, it is returned unchanged.
// If the content exceeds the limit, on_exceed determines the behavior:
//   - "error": returns an error
//   - "warn":  prints a warning to stderr and returns the content unchanged
//   - "truncate": truncates the content to MaxFileSize
//   - "skip":  returns shouldWrite=false
//   - "bash:<script_path>": runs the script with the content as input, uses stdout as new content
func CheckAndApplyLimit(content string, limit *CategoryLimit, entityID string, rootDir string) (string, bool, error) {
	if limit == nil || limit.MaxFileSize == 0 {
		return content, true, nil
	}

	if len(content) <= limit.MaxFileSize {
		return content, true, nil
	}

	onExceed := limit.OnExceed
	if onExceed == "" {
		onExceed = "error"
	}

	switch {
	case onExceed == "error":
		return content, false, fmt.Errorf(
			"file size %d exceeds limit %d for entity '%s'",
			len(content), limit.MaxFileSize, entityID,
		)

	case onExceed == "warn":
		fmt.Fprintf(os.Stderr, "WARNING: file size %d exceeds limit %d for entity '%s'\n",
			len(content), limit.MaxFileSize, entityID)
		return content, true, nil

	case onExceed == "truncate":
		return content[:limit.MaxFileSize], true, nil

	case onExceed == "skip":
		return "", false, nil

	case strings.HasPrefix(onExceed, "bash:"):
		return executeBashScript(content, onExceed, entityID, rootDir)

	default:
		// Unknown on_exceed value: treat as error
		return content, false, fmt.Errorf(
			"file size %d exceeds limit %d for entity '%s' (unknown on_exceed: %s)",
			len(content), limit.MaxFileSize, entityID, onExceed,
		)
	}
}

// executeBashScript runs a user-defined script to process oversized content.
func executeBashScript(content, onExceed, entityID, rootDir string) (string, bool, error) {
	scriptPath := strings.TrimPrefix(onExceed, "bash:")
	absScript := filepath.Join(rootDir, scriptPath)

	// Check script existence
	if _, err := os.Stat(absScript); os.IsNotExist(err) {
		return content, false, fmt.Errorf(
			"on_exceed script not found: %s (for entity '%s')",
			absScript, entityID,
		)
	}

	// Write content to temp file
	tmpFile, err := createTempFile(content)
	if err != nil {
		return content, false, fmt.Errorf(
			"failed to create temp file for on_exceed script (entity '%s'): %w",
			entityID, err,
		)
	}
	defer os.Remove(tmpFile)

	// Execute script
	bashBin := resolveBashPath()
	cmd := exec.Command(bashBin, absScript, "--prompt", tmpFile)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return content, false, fmt.Errorf(
			"on_exceed script failed for entity '%s': %w",
			entityID, err,
		)
	}

	return string(output), true, nil
}

// createTempFile writes content to a temporary file and returns its path.
func createTempFile(content string) (string, error) {
	hash := randomHex(16)
	tmpFile := filepath.Join(os.TempDir(), hash+".txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return "", err
	}
	return tmpFile, nil
}

// randomHex returns a hex string of n random bytes.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// FindTarget searches resolved manifest for a target entity with the given ID.
func FindTarget(resolved *manifest.ResolvedManifest, targetID string) *manifest.Entity {
	for _, target := range resolved.Entities["target"] {
		if target.ID == targetID {
			return target
		}
	}
	return nil
}

// resolveBashPath resolves the path to the bash executable.
// On Windows, bash may not be in PATH, so we fall back to known Git Bash locations.
func resolveBashPath() string {
	if path, err := exec.LookPath("bash"); err == nil {
		return path
	}
	// Windows: try common Git Bash locations
	candidates := []string{
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files (x86)\Git\bin\bash.exe`,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "bash" // fallback
}
