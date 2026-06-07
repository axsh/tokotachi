package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// CursorEmitter emits resolved manifest entities to Cursor IDE's .cursor/ directory.
type CursorEmitter struct {
	RootDir string
}

// NewCursorEmitter creates a new CursorEmitter.
func NewCursorEmitter(rootDir string) *CursorEmitter {
	return &CursorEmitter{RootDir: rootDir}
}

// CursorRuleFrontmatter is the YAML frontmatter for .mdc rule files.
type CursorRuleFrontmatter struct {
	Description string   `yaml:"description"`
	Globs       []string `yaml:"globs,omitempty"`
	AlwaysApply bool     `yaml:"alwaysApply"`
}

// resolvePaths returns the target rules and skills directory paths.
func (c *CursorEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string) {
	rulesPath := ".cursor/rules/"
	skillsPath := ".cursor/skills/"

	// Extract overrides from target cursor entity
	for _, target := range resolved.Entities["target"] {
		if target.ID == "cursor" {
			if paths, ok := target.Raw["paths"].(map[string]any); ok {
				if r, ok := paths["rules"].(string); ok {
					rulesPath = r
				}
				if s, ok := paths["skills"].(string); ok {
					skillsPath = s
				}
			}
		}
	}

	if apply {
		return filepath.Join(c.RootDir, rulesPath),
			filepath.Join(c.RootDir, skillsPath)
	}

	return filepath.Join(buildDir, "cursor", rulesPath),
		filepath.Join(buildDir, "cursor", skillsPath)
}

func (c *CursorEmitter) Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) error {
	rulesDir, skillsDir := c.resolvePaths(resolved, buildDir, apply)

	// Track emitted files for immune mode orphan cleanup
	emittedFiles := make(map[string]bool)

	// Extract size limits from the cursor target entity
	limits := ExtractLimits(FindTarget(resolved, "cursor"))

	// 1. Emit Policies as .mdc files
	for _, policy := range resolved.Entities["policy"] {
		filename := policy.ID + ".mdc"

		var body string
		if b, ok := policy.Raw["body"].(string); ok {
			body = b
		}

		body = stripFrontmatter(body)
		body = strings.ReplaceAll(body, "\r\n", "\n")

		var mode string
		if activation, ok := policy.Raw["activation"].(map[string]any); ok {
			mode, _ = activation["mode"].(string)
		}

		fm := CursorRuleFrontmatter{
			Description: policy.Title,
			AlwaysApply: mode == "always",
		}

		// Convert paths to globs
		if rawPaths, ok := policy.Raw["paths"].([]any); ok {
			for _, p := range rawPaths {
				if pStr, ok := p.(string); ok {
					fm.Globs = append(fm.Globs, pStr)
				}
			}
		}

		fmBytes, err := yaml.Marshal(fm)
		if err != nil {
			return fmt.Errorf("failed to marshal cursor rule frontmatter for %s: %w", policy.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for rules
		if limits != nil && limits.Rules != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Rules, policy.ID, c.RootDir)
			if limitErr != nil {
				return limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(rulesDir, filename)
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}

	// NOTE: resolved.Entities["skip"] is intentionally NOT emitted.
	// Skip entities are documentation-only and not deployed to agent targets.

	// 2. Emit Capabilities as SKILL.md (same format as Antigravity skills)
	for _, skill := range resolved.Entities["capability"] {
		var body string
		if b, ok := skill.Raw["body"].(string); ok {
			body = b
		}
		body = stripFrontmatter(body)
		body = strings.ReplaceAll(body, "\r\n", "\n")

		desc, _ := skill.Raw["description"].(string)
		manualOnly, _ := skill.Raw["manual_only"].(bool)

		fm := SkillFrontmatter{
			Name:                   skill.ID,
			Description:            desc,
			DisableModelInvocation: manualOnly,
		}

		fmBytes, err := yaml.Marshal(fm)
		if err != nil {
			return fmt.Errorf("failed to marshal cursor skill frontmatter for %s: %w", skill.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for skills
		if limits != nil && limits.Skills != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Skills, skill.ID, c.RootDir)
			if limitErr != nil {
				return limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(skillsDir, skill.ID, "SKILL.md")
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}

	// 3. Emit Procedures as Skills (Cursor has no workflow equivalent,
	//    but skills serve as on-demand procedure guides)
	for _, proc := range resolved.Entities["procedure"] {
		var body string
		if b, ok := proc.Raw["body"].(string); ok {
			body = b
		} else if steps, ok := proc.Raw["steps"].([]any); ok {
			var sb strings.Builder
			sb.WriteString("# ")
			sb.WriteString(proc.Title)
			sb.WriteString("\n\n## Steps\n\n")
			for i, step := range steps {
				sb.WriteString(fmt.Sprintf("%d. %v\n", i+1, step))
			}
			body = sb.String()
		}
		body = stripFrontmatter(body)
		body = strings.ReplaceAll(body, "\r\n", "\n")

		manualOnly := false
		if trigger, ok := proc.Raw["trigger"].(map[string]any); ok {
			manualOnly, _ = trigger["manual_only"].(bool)
		}

		fm := SkillFrontmatter{
			Name:                   proc.ID,
			Description:            proc.Title,
			DisableModelInvocation: manualOnly,
		}

		fmBytes, err := yaml.Marshal(fm)
		if err != nil {
			return fmt.Errorf(
				"failed to marshal procedure-as-skill frontmatter for %s: %w",
				proc.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for skills (procedures emit as skills in Cursor)
		if limits != nil && limits.Skills != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Skills, proc.ID, c.RootDir)
			if limitErr != nil {
				return limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(skillsDir, proc.ID, "SKILL.md")
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}

	// 4. Immune mode: clean orphan files from target directories
	if opts.Mode == EmitModeImmune {
		targetDirs := []string{rulesDir, skillsDir}
		if _, err := CleanOrphanFiles(targetDirs, emittedFiles, opts.DryRun); err != nil {
			return fmt.Errorf("failed to clean orphan files: %w", err)
		}
	}

	return nil
}

// Check verifies if generated files in project paths match the resolved manifest.
func (c *CursorEmitter) Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error) {
	// 1. Generate expected output in a temporary buildDir structure
	tempBuildDir, err := os.MkdirTemp("", "agentctl-cursor-check")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for emit check: %w", err)
	}
	defer os.RemoveAll(tempBuildDir)

	if err := c.Emit(resolved, tempBuildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		return false, fmt.Errorf("failed to dry-run emit: %w", err)
	}

	// 2. Compute expected files
	expectedBaseDir := filepath.Join(tempBuildDir, "cursor")
	expectedFiles := make(map[string]string) // relative path -> expected content
	err = filepath.Walk(expectedBaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(expectedBaseDir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		expectedFiles[rel] = string(data)
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("failed to walk expected output: %w", err)
	}

	// 3. Get paths in the live project directory
	rulesDirLive, skillsDirLive := c.resolvePaths(resolved, buildDir, true)
	liveDirs := map[string]string{
		"rules":  rulesDirLive,
		"skills": skillsDirLive,
	}

	// Track all files present in the live target directories
	liveFilesFound := make(map[string]bool)
	hasDrift := false

	// Compare expected files with live files
	for relPath, expectedContent := range expectedFiles {
		var livePath string
		matched := false

		for cat, liveDir := range liveDirs {
			folderName := ".cursor/" + cat + "/"
			for _, target := range resolved.Entities["target"] {
				if target.ID == "cursor" {
					if paths, ok := target.Raw["paths"].(map[string]any); ok {
						if folder, ok := paths[cat].(string); ok {
							folderName = folder
						}
					}
				}
			}

			folderPrefix := filepath.ToSlash(folderName)
			relToSlash := filepath.ToSlash(relPath)
			if strings.HasPrefix(relToSlash, folderPrefix) {
				subPath := strings.TrimPrefix(relToSlash, folderPrefix)
				livePath = filepath.Join(liveDir, filepath.FromSlash(subPath))
				matched = true
				break
			}
		}

		if !matched {
			livePath = filepath.Join(c.RootDir, relPath)
		}

		liveFilesFound[filepath.Clean(livePath)] = true

		liveData, err := os.ReadFile(livePath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "DRIFT: target file %s does not exist\n", livePath)
				hasDrift = true
				continue
			}
			return false, fmt.Errorf("failed to read target file %s: %w", livePath, err)
		}

		liveContent := strings.ReplaceAll(string(liveData), "\r\n", "\n")
		expectedNorm := strings.ReplaceAll(expectedContent, "\r\n", "\n")

		if liveContent != expectedNorm {
			fmt.Fprintf(os.Stderr, "DRIFT: target file %s is out of date\n", livePath)
			d := simpleDiff(livePath, liveContent, expectedNorm)
			fmt.Fprint(os.Stderr, d)
			hasDrift = true
		}
	}

	// 4. Scan live target directories for untracked files
	for _, liveDir := range liveDirs {
		if _, err := os.Stat(liveDir); os.IsNotExist(err) {
			continue
		}
		err = filepath.Walk(liveDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			// Check markdown (.md) and cursor rule (.mdc) files
			lower := strings.ToLower(info.Name())
			if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".mdc") {
				return nil
			}
			if info.Name() == "README.md" || info.Name() == ".gitkeep" {
				return nil
			}

			cleanPath := filepath.Clean(path)
			if !liveFilesFound[cleanPath] {
				fmt.Fprintf(os.Stderr, "DRIFT: untracked file %s found in target directory\n", path)
				hasDrift = true
			}
			return nil
		})
		if err != nil {
			return false, fmt.Errorf("failed to walk live target directory %s: %w", liveDir, err)
		}
	}

	return !hasDrift, nil
}
