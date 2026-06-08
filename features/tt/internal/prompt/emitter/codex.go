package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

// CodexEmitter emits resolved manifest entities to .agents/ directory
// and manages the AGENTS.md marker section.
type CodexEmitter struct {
	RootDir string
}

// NewCodexEmitter creates a new CodexEmitter.
func NewCodexEmitter(rootDir string) *CodexEmitter {
	return &CodexEmitter{RootDir: rootDir}
}

// resolvePaths returns the target rules and skills directory paths.
func (c *CodexEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string) {
	rulesPath := ".agents/rules/"
	skillsPath := ".agents/skills/"

	// Extract overrides from target codex entity
	for _, target := range resolved.Entities["target"] {
		if target.ID == "codex" {
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

	return filepath.Join(buildDir, "codex", rulesPath),
		filepath.Join(buildDir, "codex", skillsPath)
}

func (c *CodexEmitter) Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) (*EmitResult, error) {
	rulesDir, skillsDir := c.resolvePaths(resolved, buildDir, apply)

	// Track emitted files for immune mode orphan cleanup
	emittedFiles := make(map[string]bool)

	// Extract size limits from the codex target entity
	limits := ExtractLimits(FindTarget(resolved, "codex"))

	// Track emitted entity data for AGENTS.md marker content
	var emittedPolicies []*manifest.Entity
	var skillIDs []string

	// 1. Emit Policies as pure Markdown (no frontmatter)
	for _, policy := range resolved.Entities["policy"] {
		filename := policy.ID + ".md"

		var body string
		if b, ok := policy.Raw["body"].(string); ok {
			body = b
		}

		body = stripFrontmatter(body)
		body = strings.ReplaceAll(body, "\r\n", "\n")

		// Codex: emit without any frontmatter
		content := body

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
		emittedPolicies = append(emittedPolicies, policy)
	}

	// NOTE: resolved.Entities["skip"] is intentionally NOT emitted.

	// 2. Emit Capabilities as SKILL.md
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
			return nil, fmt.Errorf("failed to marshal codex skill frontmatter for %s: %w", skill.ID, err)
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
		skillIDs = append(skillIDs, skill.ID)
	}

	// 3. Emit Procedures as Skills
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
		skillIDs = append(skillIDs, proc.ID)
	}

	// 4. Update index file marker section (if configured)
	indexFile := c.resolveIndexFile(resolved)
	if indexFile != "" {
		markerContent := c.generateMarkerContent(emittedPolicies, skillIDs)
		if apply {
			agentsPath := filepath.Join(c.RootDir, indexFile)
			if err := ReplaceMarkerSection(agentsPath, markerContent); err != nil {
				return nil, fmt.Errorf("failed to update %s marker section: %w", indexFile, err)
			}
		} else {
			// Write marker content for check comparison
			markerPath := filepath.Join(buildDir, "codex", indexFile+".marker")
			dir := filepath.Dir(markerPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
			if err := os.WriteFile(markerPath, []byte(markerContent), 0644); err != nil {
				return nil, fmt.Errorf("failed to write marker content: %w", err)
			}
		}
	}

	// 5. Return EmitResult for coordinated orphan cleanup in deploy pipeline
	return &EmitResult{
		EmittedFiles: emittedFiles,
		TargetDirs:   []string{rulesDir, skillsDir},
	}, nil
}



// generateMarkerContent generates the content for the AGENT-MANAGED section
// in the index file, listing emitted rules and skills with optional applies_when guidance.
func (c *CodexEmitter) generateMarkerContent(policies []*manifest.Entity, skillIDs []string) string {
	// Resolve paths for the rules and skills directories
	rulesPath := ".agents/rules/"
	skillsPath := ".agents/skills/"

	var sb strings.Builder
	sb.WriteString("\n## Project Guidelines\n\n")
	sb.WriteString("This project follows structured rules and workflows managed under `.agents/`.\n\n")

	// Rules section
	sb.WriteString("### Rules\n")
	sb.WriteString("For detailed project rules, read the following documents:\n")

	sorted := make([]*manifest.Entity, len(policies))
	copy(sorted, policies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	for _, p := range sorted {
		line := fmt.Sprintf("- `%s%s.md`", rulesPath, p.ID)
		if aw, ok := p.Raw["applies_when"].(string); ok && aw != "" {
			line += " - " + aw
		}
		sb.WriteString(line + "\n")
	}

	// Skills section
	sb.WriteString("\n### Skills\n")
	sb.WriteString("Reusable workflows are available as skills under `" + skillsPath + "`.\n")

	return sb.String()
}

// resolveIndexFile extracts the index_file path from the codex target entity.
// Returns empty string if not set.
func (c *CodexEmitter) resolveIndexFile(resolved *manifest.ResolvedManifest) string {
	target := FindTarget(resolved, "codex")
	if target == nil {
		return ""
	}
	if indexFile, ok := target.Raw["index_file"].(string); ok {
		return indexFile
	}
	return ""
}

// Check verifies if generated files in project paths match the resolved manifest.
func (c *CodexEmitter) Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error) {
	// 1. Generate expected output in a temporary buildDir structure
	tempBuildDir, err := os.MkdirTemp("", "agentctl-codex-check")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for emit check: %w", err)
	}
	defer os.RemoveAll(tempBuildDir)

	if _, err := c.Emit(resolved, tempBuildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		return false, fmt.Errorf("failed to dry-run emit: %w", err)
	}

	// 2. Compute expected files (excluding AGENTS.md.marker)
	expectedBaseDir := filepath.Join(tempBuildDir, "codex")
	expectedFiles := make(map[string]string) // relative path -> expected content
	err = filepath.Walk(expectedBaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Skip the marker file; marker is checked separately
		if info.Name() == "AGENTS.md.marker" {
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

	liveFilesFound := make(map[string]bool)
	hasDrift := false

	// Compare expected files with live files
	for relPath, expectedContent := range expectedFiles {
		var livePath string
		matched := false

		for cat, liveDir := range liveDirs {
			folderName := ".agents/" + cat + "/"
			for _, target := range resolved.Entities["target"] {
				if target.ID == "codex" {
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

	// 5. Check index file marker section (if configured)
	indexFile := c.resolveIndexFile(resolved)
	if indexFile != "" {
		expectedMarkerPath := filepath.Join(tempBuildDir, "codex", indexFile+".marker")
		expectedMarkerData, err := os.ReadFile(expectedMarkerPath)
		if err != nil {
			return false, fmt.Errorf("failed to read expected marker content: %w", err)
		}
		expectedMarker := strings.ReplaceAll(string(expectedMarkerData), "\r\n", "\n")

		agentsPath := filepath.Join(c.RootDir, indexFile)
		agentsData, err := os.ReadFile(agentsPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "DRIFT: %s does not exist\n", indexFile)
				hasDrift = true
				return !hasDrift, nil
			}
			return false, fmt.Errorf("failed to read %s: %w", indexFile, err)
		}
		agentsContent := strings.ReplaceAll(string(agentsData), "\r\n", "\n")

		currentMarker, found := ExtractMarkerSection(agentsContent)
		if !found {
			fmt.Fprintf(os.Stderr, "DRIFT: %s has no AGENT-MANAGED marker section\n", indexFile)
			hasDrift = true
		} else {
			expectedInner := expectedMarker
			currentInner := currentMarker

			warningLines := []string{
				"<!-- WARNING: This section is auto-generated. Do not edit manually. -->\n",
				"<!-- Changes between AGENT-MANAGED:BEGIN and AGENT-MANAGED:END will be overwritten. -->\n",
				"<!-- To modify, update source files in prompts/manifest/ and re-run the deploy command. -->\n",
			}
			for _, wl := range warningLines {
				currentInner = strings.Replace(currentInner, wl, "", 1)
			}

			if strings.TrimSpace(currentInner) != strings.TrimSpace(expectedInner) {
				fmt.Fprintf(os.Stderr, "DRIFT: %s marker section is out of date\n", indexFile)
				hasDrift = true
			}
		}
	}

	return !hasDrift, nil
}
