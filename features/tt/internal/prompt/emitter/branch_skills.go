package emitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BranchSkill represents a skill file discovered in branches/*/skills/.
type BranchSkill struct {
	// ID is the skill directory name (e.g., "__far-knowledge-error-handling")
	ID string
	// Path is the absolute path to the SKILL.md file
	Path string
	// Content is the raw content of the SKILL.md file
	Content string
	// BranchID is the branch directory name where this skill was found
	BranchID string
}

// ScanBranchSkills scans prompts/memory/branches/*/skills/ directories
// for far-knowledge skills and returns them as BranchSkill entries.
// Skills are identified by having a SKILL.md file in a subdirectory.
func ScanBranchSkills(rootDir string) ([]BranchSkill, error) {
	branchesDir := filepath.Join(rootDir, "prompts", "memory", "branches")

	if _, err := os.Stat(branchesDir); os.IsNotExist(err) {
		return nil, nil
	}

	branchEntries, err := os.ReadDir(branchesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read branches directory: %w", err)
	}

	var skills []BranchSkill

	for _, branchEntry := range branchEntries {
		if !branchEntry.IsDir() {
			continue
		}

		skillsDir := filepath.Join(branchesDir, branchEntry.Name(), "skills")
		if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
			continue
		}

		skillEntries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue // skip unreadable dirs
		}

		for _, skillEntry := range skillEntries {
			if !skillEntry.IsDir() {
				continue
			}

			skillMDPath := filepath.Join(skillsDir, skillEntry.Name(), "SKILL.md")
			content, err := os.ReadFile(skillMDPath)
			if err != nil {
				continue // skip skills without SKILL.md
			}

			skills = append(skills, BranchSkill{
				ID:       skillEntry.Name(),
				Path:     skillMDPath,
				Content:  string(content),
				BranchID: branchEntry.Name(),
			})
		}
	}

	return skills, nil
}

// EmitBranchSkills writes branch skills to the skills directory.
// It writes each skill as skillsDir/<skill-id>/SKILL.md.
func EmitBranchSkills(branchSkills []BranchSkill, skillsDir string, opts EmitOptions) (map[string]bool, error) {
	emitted := make(map[string]bool)

	for _, skill := range branchSkills {
		// Normalize line endings
		content := strings.ReplaceAll(skill.Content, "\r\n", "\n")

		outputPath := filepath.Join(skillsDir, skill.ID, "SKILL.md")
		if err := writeFileWithMode(outputPath, content, opts.Mode); err != nil {
			return nil, fmt.Errorf("failed to write branch skill %s: %w", skill.ID, err)
		}
		emitted[filepath.Clean(outputPath)] = true
	}

	return emitted, nil
}
