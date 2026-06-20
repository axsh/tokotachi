package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

type AntigravityEmitter struct {
	RootDir string
}

func NewAntigravityEmitter(rootDir string) *AntigravityEmitter {
	return &AntigravityEmitter{RootDir: rootDir}
}

type SkillFrontmatter struct {
	Name                   string   `yaml:"name"`
	Description            string   `yaml:"description"`
	Paths                  []string `yaml:"paths,omitempty"`
	DisableModelInvocation bool     `yaml:"disable-model-invocation"`
}

// WorkflowFrontmatter defines the YAML frontmatter for workflow files.
type WorkflowFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}


// resolvePaths returns the target rules, skills, and workflows directory paths.
func (a *AntigravityEmitter) resolvePaths(resolved *manifest.ResolvedManifest, buildDir string, apply bool) (string, string, string) {
	rulesPath := ".agent/rules/"
	skillsPath := ".agent/skills/"
	workflowsPath := ".agent/workflows/"

	// Extract overrides from target antigravity entity
	for _, target := range resolved.Entities["target"] {
		if target.ID == "antigravity" {
			if paths, ok := target.Raw["paths"].(map[string]any); ok {
				if r, ok := paths["rules"].(string); ok {
					rulesPath = r
				}
				if s, ok := paths["skills"].(string); ok {
					skillsPath = s
				}
				if w, ok := paths["workflows"].(string); ok {
					workflowsPath = w
				}
			}
		}
	}

	if apply {
		return filepath.Join(a.RootDir, rulesPath),
			filepath.Join(a.RootDir, skillsPath),
			filepath.Join(a.RootDir, workflowsPath)
	}

	return filepath.Join(buildDir, "antigravity", rulesPath),
		filepath.Join(buildDir, "antigravity", skillsPath),
		filepath.Join(buildDir, "antigravity", workflowsPath)
}

func (a *AntigravityEmitter) Emit(resolved *manifest.ResolvedManifest, buildDir string, apply bool, opts EmitOptions) (*EmitResult, error) {
	// Clean the emitter's buildDir subdirectory to ensure compiled output matches current templates
	if err := CleanTargetDirs(filepath.Join(buildDir, "antigravity")); err != nil {
		return nil, fmt.Errorf("failed to clean build dir for antigravity: %w", err)
	}

	rulesDir, skillsDir, workflowsDir := a.resolvePaths(resolved, buildDir, apply)

	// Track emitted files for immune mode orphan cleanup
	emittedFiles := make(map[string]bool)

	// Build template context for resolving {{kind:id}} variables in body text
	tmplCtx := &TemplateContext{
		Paths:      a.resolveTargetPaths(resolved),
		MemBase:    resolveMemoryBase(),
		TargetName: "antigravity",
	}

	// Extract size limits and includes from the antigravity target entity
	antigravityTarget := FindTarget(resolved, "antigravity")
	limits := ExtractLimits(antigravityTarget)
	inc := ExtractIncludes(antigravityTarget)

	// 1. Emit Policies
	if inc.Policy {
	for _, policy := range resolved.Entities["policy"] {
		var filename string
		if policy.ID == "project-instructions" {
			filename = "instructions.md"
		} else {
			filename = policy.ID + ".md"
		}

		var body string
		if b, ok := policy.Raw["body"].(string); ok {
			body = b
		}

		body = stripFrontmatter(body)
		// Normalize line endings to \n
		body = strings.ReplaceAll(body, "\r\n", "\n")
		body = ResolveTemplateVars(body, tmplCtx)

		var mode string
		if activation, ok := policy.Raw["activation"].(map[string]any); ok {
			mode, _ = activation["mode"].(string)
		}

		var content string
		if mode == "always" {
			content = "---\ntrigger: always_on\n---\n\n" + body
		} else {
			content = body
		}

		// Apply size limit check for rules
		if limits != nil && limits.Rules != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Rules, policy.ID, a.RootDir)
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
	// Skip entities are documentation-only and not deployed to agent targets.

	// 2. Emit Capabilities (Skills)
	if inc.Capability {
	for _, skill := range resolved.Entities["capability"] {
		var body string
		if b, ok := skill.Raw["body"].(string); ok {
			body = b
		}
		body = stripFrontmatter(body)
		body = strings.ReplaceAll(body, "\r\n", "\n")
		body = ResolveTemplateVars(body, tmplCtx)

		desc, _ := skill.Raw["description"].(string)
		manualOnly, _ := skill.Raw["manual_only"].(bool)

		var paths []string
		if rawPaths, ok := skill.Raw["paths"].([]any); ok {
			for _, p := range rawPaths {
				if pStr, ok := p.(string); ok {
					paths = append(paths, pStr)
				}
			}
		}

		fm := SkillFrontmatter{
			Name:                   skill.ID,
			Description:            desc,
			Paths:                  paths,
			DisableModelInvocation: manualOnly,
		}

		fmBytes, err := yaml.Marshal(fm)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal skill frontmatter for %s: %w", skill.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for skills
		if limits != nil && limits.Skills != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Skills, skill.ID, a.RootDir)
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

	// 3. Emit Procedures (as Workflows - flat .md files)
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
		body = ResolveTemplateVars(body, tmplCtx)

		manualOnly := false
		if trigger, ok := proc.Raw["trigger"].(map[string]any); ok {
			manualOnly, _ = trigger["manual_only"].(bool)
		}

		_ = manualOnly // workflows do not use disable-model-invocation

		fm := WorkflowFrontmatter{
			Name:        proc.ID,
			Description: proc.Title,
		}

		fmBytes, err := yaml.Marshal(fm)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal workflow frontmatter for %s: %w", proc.ID, err)
		}

		content := fmt.Sprintf("---\n%s---\n\n%s", string(fmBytes), body)

		// Apply size limit check for skills (workflows share skills limits)
		if limits != nil && limits.Skills != nil {
			var shouldWrite bool
			var limitErr error
			content, shouldWrite, limitErr = CheckAndApplyLimit(content, limits.Skills, proc.ID, a.RootDir)
			if limitErr != nil {
				return nil, limitErr
			}
			if !shouldWrite {
				continue
			}
		}

		outputPath := filepath.Join(workflowsDir, proc.ID+".md")
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return nil, err
		}
		emittedFiles[filepath.Clean(outputPath)] = true
	}
	} // end if inc.Procedure

	// 4. Emit Branch Skills (far-knowledge skills from branches/*/skills/)
	branchSkills, err := ScanBranchSkills(a.RootDir)
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
		TargetDirs:   []string{rulesDir, skillsDir, workflowsDir},
	}, nil
}

// resolveTargetPaths extracts the target paths from the antigravity target entity.
func (a *AntigravityEmitter) resolveTargetPaths(resolved *manifest.ResolvedManifest) TargetPaths {
	tp := TargetPaths{
		Rules:     ".agent/rules/",
		Skills:    ".agent/skills/",
		Workflows: ".agent/workflows/",
	}
	for _, target := range resolved.Entities["target"] {
		if target.ID == "antigravity" {
			if paths, ok := target.Raw["paths"].(map[string]any); ok {
				if r, ok := paths["rules"].(string); ok {
					tp.Rules = ensureTrailingSlash(r)
				}
				if s, ok := paths["skills"].(string); ok {
					tp.Skills = ensureTrailingSlash(s)
				}
				if w, ok := paths["workflows"].(string); ok {
					tp.Workflows = ensureTrailingSlash(w)
				}
			}
		}
	}
	return tp
}

// resolveMemoryBase returns the base directory for memory documents.
func resolveMemoryBase() string {
	return "prompts/memory"
}

func (a *AntigravityEmitter) Check(resolved *manifest.ResolvedManifest, buildDir string) (bool, error) {
	// 1. Generate expected output in a temporary buildDir structure
	tempBuildDir, err := os.MkdirTemp("", "agentctl-emit-check")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir for emit check: %w", err)
	}
	defer os.RemoveAll(tempBuildDir)

	if _, err := a.Emit(resolved, tempBuildDir, false, EmitOptions{Mode: EmitModeOverwrite}); err != nil {
		return false, fmt.Errorf("failed to dry-run emit: %w", err)
	}

	// 2. Compute expected files
	rulesDirExpected := filepath.Join(tempBuildDir, "antigravity")
	// Get all files generated under tempBuildDir/antigravity
	expectedFiles := make(map[string]string) // relative path under targets -> expected content
	err = filepath.Walk(rulesDirExpected, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(rulesDirExpected, path)
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
	rulesDir, skillsDir, workflowsDir := a.resolvePaths(resolved, buildDir, true)
	liveDirs := map[string]string{
		"rules":     rulesDir,
		"skills":    skillsDir,
		"workflows": workflowsDir,
	}

	// Track all markdown files present in the live target directories
	liveFilesFound := make(map[string]bool)
	hasDrift := false

	// Compare expected files with live files
	for relPath, expectedContent := range expectedFiles {
		// Figure out where this relPath belongs in the live project structure.
		// relPath starts with the directory prefix defined in targets, e.g., "rules_dir/instructions.md"
		// Let's identify which category it belongs to based on the target config.
		// Since relPath is relative to the tempBuildDir/antigravity/ directory,
		// it contains the target config folder as its first component.
		var livePath string
		matched := false

		for cat, liveDir := range liveDirs {
			// Find override folder name from target definition
			folderName := ".agent/" + cat + "/"
			for _, target := range resolved.Entities["target"] {
				if target.ID == "antigravity" {
					if paths, ok := target.Raw["paths"].(map[string]any); ok {
						if folder, ok := paths[cat].(string); ok {
							folderName = folder
						}
					}
				}
			}

			// Normalize folderName to use slash for checking prefix
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
			// Fallback: join relPath directly with root dir
			livePath = filepath.Join(a.RootDir, relPath)
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
			// simple diff
			d := simpleDiff(livePath, liveContent, expectedNorm)
			fmt.Fprint(os.Stderr, d)
			hasDrift = true
		}
	}

	// 4. Scan live target directories for untracked markdown files
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
			// Only check markdown files (.md)
			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}
			// Ignore standard non-generated files like .gitkeep or README.md if they exist, but generally markdown files are target rules/workflows/skills.
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

// writeFile writes content to a file, creating parent directories if needed.
func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// simpleDiff produces a simple line-based diff between current and expected content.
func simpleDiff(label, current, expected string) string {
	currentLines := strings.Split(current, "\n")
	expectedLines := strings.Split(expected, "\n")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- %s (current)\n", label))
	b.WriteString(fmt.Sprintf("+++ %s (expected)\n", label))

	maxLen := len(currentLines)
	if len(expectedLines) > maxLen {
		maxLen = len(expectedLines)
	}

	for i := range maxLen {
		var curLine, expLine string
		if i < len(currentLines) {
			curLine = currentLines[i]
		}
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if curLine != expLine {
			if i < len(currentLines) {
				b.WriteString(fmt.Sprintf("- %s\n", curLine))
			}
			if i < len(expectedLines) {
				b.WriteString(fmt.Sprintf("+ %s\n", expLine))
			}
		}
	}

	return b.String()
}

func stripFrontmatter(content string) string {
	contentNorm := strings.ReplaceAll(content, "\r\n", "\n")
	if strings.HasPrefix(contentNorm, "---\n") {
		parts := strings.SplitN(contentNorm, "---\n", 3)
		if len(parts) >= 3 {
			return strings.TrimSpace(parts[2]) + "\n"
		}
	}
	return content
}
