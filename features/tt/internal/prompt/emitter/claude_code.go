package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// ClaudeCodeEmitter emits resolved manifest entities to .claude/ directory.
type ClaudeCodeEmitter struct {
	RootDir string
}

// NewClaudeCodeEmitter creates a new ClaudeCodeEmitter.
func NewClaudeCodeEmitter(rootDir string) *ClaudeCodeEmitter {
	return &ClaudeCodeEmitter{RootDir: rootDir}
}

// ClaudeCodeRuleFrontmatter is the YAML frontmatter for .claude/rules/*.md.
// Claude Code rules use only "paths" for scoping; no alwaysApply flag.
type ClaudeCodeRuleFrontmatter struct {
	Paths []string `yaml:"paths,omitempty"`
}

// resolvePaths returns the target rules and skills directory paths.
func (c *ClaudeCodeEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string) {
	rulesPath := ".claude/rules/"
	skillsPath := ".claude/skills/"

	// Extract overrides from target claude-code entity
	for _, target := range resolved.Entities["target"] {
		if target.ID == "claude-code" {
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

	return filepath.Join(buildDir, "claude-code", rulesPath),
		filepath.Join(buildDir, "claude-code", skillsPath)
}

func (c *ClaudeCodeEmitter) Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) (*EmitResult, error) {
	// Clean the emitter's buildDir subdirectory to ensure compiled output matches current templates
	if err := CleanTargetDirs(filepath.Join(buildDir, "claude-code")); err != nil {
		return nil, fmt.Errorf("failed to clean build dir for claude-code: %w", err)
	}

	rulesDir, skillsDir := c.resolvePaths(resolved, buildDir, apply)

	// Track emitted files for immune mode orphan cleanup
	emittedFiles := make(map[string]bool)

	// Extract size limits and includes from the claude-code target entity
	claudeTarget := FindTarget(resolved, "claude-code")
	limits := ExtractLimits(claudeTarget)
	inc := ExtractIncludes(claudeTarget)

	// 1. Emit Policies as .md files
	if inc.Policy {
	for _, policy := range resolved.Entities["policy"] {
		filename := policy.ID + ".md"

		var body string
		if b, ok := policy.Raw["body"].(string); ok {
			body = b
		}

		body = stripFrontmatter(body)
		body = strings.ReplaceAll(body, "\r\n", "\n")

		// Extract paths for frontmatter
		var paths []string
		if rawPaths, ok := policy.Raw["paths"].([]any); ok {
			for _, p := range rawPaths {
				if pStr, ok := p.(string); ok {
					paths = append(paths, pStr)
				}
			}
		}

		var content string
		if len(paths) > 0 {
			// Has paths -> emit with frontmatter
			fm := ClaudeCodeRuleFrontmatter{Paths: paths}
			fmBytes, err := yaml.Marshal(fm)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal claude-code rule frontmatter for %s: %w", policy.ID, err)
			}
			content = fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)
		} else {
			// No paths -> emit without frontmatter
			content = body
		}

		// Apply size limit check for rules
		if limits != nil && limits.Rules != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Rules, policy.ID, c.RootDir)
			if limitErr != nil {
				return nil, limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(rulesDir, filename)
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return nil, err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}
	} // end if inc.Policy

	// NOTE: resolved.Entities["skip"] is intentionally NOT emitted.

	// 2. Emit Capabilities as SKILL.md
	if inc.Capability {
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
			return nil, fmt.Errorf("failed to marshal claude-code skill frontmatter for %s: %w", skill.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for skills
		if limits != nil && limits.Skills != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Skills, skill.ID, c.RootDir)
			if limitErr != nil {
				return nil, limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(skillsDir, skill.ID, "SKILL.md")
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return nil, err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}
	} // end if inc.Capability

	// 3. Emit Procedures as Skills
	if inc.Procedure {
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
			return nil, fmt.Errorf(
				"failed to marshal procedure-as-skill frontmatter for %s: %w",
				proc.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for skills
		if limits != nil && limits.Skills != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Skills, proc.ID, c.RootDir)
			if limitErr != nil {
				return nil, limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(skillsDir, proc.ID, "SKILL.md")
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return nil, err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}
	} // end if inc.Procedure

	// 4. Emit Branch Skills (far-knowledge skills from branches/*/skills/)
	branchSkills, err := ScanBranchSkills(c.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan branch skills: %w", err)
	}
	branchEmitted, err := EmitBranchSkills(branchSkills, skillsDir, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to emit branch skills: %w", err)
	}
	for k, v := range branchEmitted {
		emittedFiles[k] = v
	}

	// 5. Return EmitResult for coordinated orphan cleanup in deploy pipeline
	return &EmitResult{
		EmittedFiles: emittedFiles,
		TargetDirs:   []string{rulesDir, skillsDir},
	}, nil
}

// Check verifies if generated files in project paths match the resolved manifest.
func (c *ClaudeCodeEmitter) Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error) {
	// 1. Generate expected output in a temporary buildDir structure
	tempBuildDir, err := os.MkdirTemp("", "agentctl-claude-code-check")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for emit check: %w", err)
	}
	defer os.RemoveAll(tempBuildDir)

	if _, err := c.Emit(resolved, tempBuildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		return false, fmt.Errorf("failed to dry-run emit: %w", err)
	}

	// 2. Compute expected files
	expectedBaseDir := filepath.Join(tempBuildDir, "claude-code")
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
			folderName := ".claude/" + cat + "/"
			for _, target := range resolved.Entities["target"] {
				if target.ID == "claude-code" {
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
			lower := strings.ToLower(info.Name())
			if !strings.HasSuffix(lower, ".md") {
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
