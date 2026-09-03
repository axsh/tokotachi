package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BundleEntry is one file to copy into an emitted skill folder.
type BundleEntry struct {
	Src  string // workspace-relative, slash-normalized
	Dest string // skill-root-relative
}

// ParseBundleEntries reads capability Raw["bundle"] into typed entries.
// nil / absent yields an empty slice.
func ParseBundleEntries(raw any) ([]BundleEntry, error) {
	if raw == nil {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("bundle: expected array, got %T", raw)
	}
	out := make([]BundleEntry, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("bundle[%d]: expected object, got %T", i, item)
		}
		src, _ := m["src"].(string)
		dest, _ := m["dest"].(string)
		src = strings.TrimSpace(src)
		dest = strings.TrimSpace(dest)
		if src == "" || dest == "" {
			return nil, fmt.Errorf("bundle[%d]: src and dest are required", i)
		}
		out = append(out, BundleEntry{
			Src:  filepath.ToSlash(src),
			Dest: dest,
		})
	}
	return out, nil
}

// ValidateSkillDest ensures dest stays inside the skill root.
func ValidateSkillDest(dest string) error {
	if strings.TrimSpace(dest) == "" {
		return fmt.Errorf("bundle dest is empty")
	}
	if filepath.IsAbs(dest) {
		return fmt.Errorf("bundle dest must be relative to skill root, got absolute path %q", dest)
	}
	cleaned := filepath.Clean(dest)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("bundle dest escapes skill root: %q", dest)
	}
	// Also reject Windows-style and slash forms after Clean with ToSlash
	slash := filepath.ToSlash(cleaned)
	if slash == ".." || strings.HasPrefix(slash, "../") {
		return fmt.Errorf("bundle dest escapes skill root: %q", dest)
	}
	for _, part := range strings.Split(slash, "/") {
		if part == ".." {
			return fmt.Errorf("bundle dest escapes skill root: %q", dest)
		}
	}
	return nil
}

// RewriteBundlePaths replaces workspace-relative src paths in body with dest paths.
func RewriteBundlePaths(body string, entries []BundleEntry) string {
	if len(entries) == 0 || body == "" {
		return body
	}
	sorted := make([]BundleEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Src) > len(sorted[j].Src)
	})
	result := body
	for _, e := range sorted {
		destSlash := filepath.ToSlash(e.Dest)
		src := e.Src
		result = strings.ReplaceAll(result, "`"+src+"`", "`"+destSlash+"`")
		result = strings.ReplaceAll(result, "]("+src+")", "]("+destSlash+")")
		result = strings.ReplaceAll(result, "](./"+src+")", "](./"+destSlash+")")
	}
	return result
}

// EmitBundledFiles copies bundle entries into skillDir and returns emitted paths.
func EmitBundledFiles(skillDir, workspaceRoot string, entries []BundleEntry, opts EmitOptions) (map[string]bool, error) {
	emitted := make(map[string]bool)
	skillDirClean := filepath.Clean(skillDir)

	for _, e := range entries {
		if err := ValidateSkillDest(e.Dest); err != nil {
			return nil, err
		}
		srcPath := filepath.Join(workspaceRoot, filepath.FromSlash(e.Src))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("bundle src not found: %s: %w", e.Src, err)
		}
		outPath := filepath.Join(skillDir, filepath.FromSlash(e.Dest))
		outClean := filepath.Clean(outPath)
		rel, err := filepath.Rel(skillDirClean, outClean)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil, fmt.Errorf("bundle dest escapes skill root: %q", e.Dest)
		}
		if err := writeBytesWithMode(outClean, data, opts.Mode); err != nil {
			return nil, fmt.Errorf("failed to write bundle dest %s: %w", e.Dest, err)
		}
		emitted[outClean] = true
	}
	return emitted, nil
}

func writeBytesWithMode(path string, data []byte, mode EmitMode) error {
	if mode == EmitModeSkip {
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(os.Stderr, "SKIP: file already exists: %s\n", path)
			return nil
		}
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, data, 0644)
}
