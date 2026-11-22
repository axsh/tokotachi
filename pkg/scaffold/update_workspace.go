package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UpdateResult is the outcome of PrepareUpdateWorkspace.
type UpdateResult struct {
	WorkDir       string
	CleanApplied  bool
	Instruction   string
	RelationFiles []string
}

// UpdateWorkspaceOptions configures PrepareUpdateWorkspace.
type UpdateWorkspaceOptions struct {
	RepoRoot      string
	WorkDir       string
	LocalManifest string            // absolute path to prompts/manifest
	LatestFiles   map[string][]byte // slash-rel paths under prompts/manifest
	Stdout        io.Writer
}

// DefaultUpdateWorkDir returns tmp/scaffold-update/<category>-<name>/ under repoRoot.
func DefaultUpdateWorkDir(repoRoot, category, name string) string {
	safe := strings.ReplaceAll(category+"-"+name, "/", "-")
	return filepath.Join(repoRoot, "tmp", "scaffold-update", safe)
}

// ManifestRelPrefix is the path prefix used when collecting template files.
const ManifestRelPrefix = "prompts/manifest/"

// PrepareUpdateWorkspace builds backup/latest/instruction/doc-relations and optionally clean-applies.
func PrepareUpdateWorkspace(opts UpdateWorkspaceOptions) (*UpdateResult, error) {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.WorkDir == "" {
		return nil, fmt.Errorf("update workdir is required")
	}
	if opts.LocalManifest == "" {
		opts.LocalManifest = filepath.Join(opts.RepoRoot, "prompts", "manifest")
	}

	backupDir := filepath.Join(opts.WorkDir, "backup")
	latestDir := filepath.Join(opts.WorkDir, "latest")
	if err := os.RemoveAll(opts.WorkDir); err != nil {
		return nil, fmt.Errorf("failed to reset workdir: %w", err)
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(latestDir, 0o755); err != nil {
		return nil, err
	}

	if err := copyDirIfExists(opts.LocalManifest, backupDir); err != nil {
		return nil, fmt.Errorf("backup copy: %w", err)
	}
	for rel, data := range opts.LatestFiles {
		rel = filepath.ToSlash(rel)
		rel = strings.TrimPrefix(rel, ManifestRelPrefix)
		dest := filepath.Join(latestDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return nil, err
		}
	}

	result := &UpdateResult{WorkDir: opts.WorkDir}

	if IsCleanUpdate(backupDir, latestDir) {
		if err := replaceDir(opts.LocalManifest, latestDir); err != nil {
			return nil, fmt.Errorf("clean apply: %w", err)
		}
		result.CleanApplied = true
		msg := fmt.Sprintf("Clean update applied: local %s now matches latest (no local edits detected).\nWorkdir retained at %s for inspection.\n",
			opts.LocalManifest, opts.WorkDir)
		_, _ = io.WriteString(opts.Stdout, msg)
		return result, nil
	}

	rels, err := BuildDocRelations(backupDir, latestDir)
	if err != nil {
		return nil, err
	}
	relationFiles, err := WriteDocRelationFiles(opts.WorkDir, rels, docRelationsMaxPer)
	if err != nil {
		return nil, err
	}
	result.RelationFiles = relationFiles

	instruction := DefaultMergeInstruction(opts.WorkDir, relationFiles)
	result.Instruction = instruction
	instrPath := filepath.Join(opts.WorkDir, "instruction.md")
	if err := os.WriteFile(instrPath, []byte(instruction), 0o644); err != nil {
		return nil, err
	}

	_, _ = io.WriteString(opts.Stdout, instruction)
	_, _ = io.WriteString(opts.Stdout, fmt.Sprintf(
		"\n---\nInstruction file saved at: %s\nRelation files: %s\n",
		filepath.ToSlash(instrPath),
		strings.Join(relationFiles, ", "),
	))
	return result, nil
}

// IsCleanUpdate is true when backup and latest have the same relative file set and identical bytes.
func IsCleanUpdate(backupDir, latestDir string) bool {
	backupMap, err := readAllFiles(backupDir)
	if err != nil {
		return false
	}
	latestMap, err := readAllFiles(latestDir)
	if err != nil {
		return false
	}
	if len(backupMap) != len(latestMap) {
		return false
	}
	if len(latestMap) == 0 {
		return false
	}
	for rel, data := range latestMap {
		b, ok := backupMap[rel]
		if !ok {
			return false
		}
		if string(b) != string(data) {
			return false
		}
	}
	for rel := range backupMap {
		if _, ok := latestMap[rel]; !ok {
			return false
		}
	}
	return true
}

func readAllFiles(root string) (map[string][]byte, error) {
	out := make(map[string][]byte)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return out, nil
	}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = data
		return nil
	})
	return out, err
}

func copyDirIfExists(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	return copyTree(src, dst)
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

func replaceDir(dest, src string) error {
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return copyTree(src, dest)
}

// CollectManifestFiles filters template files whose destination involves prompts/manifest.
func CollectManifestFiles(templateFiles []DownloadedFile, placement *Placement) map[string][]byte {
	out := make(map[string][]byte)
	base := ""
	if placement != nil {
		base = filepath.ToSlash(placement.BaseDir)
	}
	for _, file := range templateFiles {
		p := filepath.ToSlash(file.RelativePath)
		full := p
		if base != "" && base != "." {
			full = strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(p, "/")
		}
		full = strings.TrimPrefix(full, "./")
		idx := strings.Index(full, ManifestRelPrefix)
		if idx < 0 {
			if strings.Contains(full, "/code_content/") || strings.HasPrefix(full, "code_content/") {
				if i := strings.Index(full, "code_content/"); i >= 0 {
					rel := ManifestRelPrefix + full[i:]
					out[rel] = file.Content
				}
			}
			continue
		}
		out[full[idx:]] = file.Content
	}
	return out
}
