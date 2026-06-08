package task

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"gopkg.in/yaml.v3"
)

// WriteKnowledgeAtom writes a Knowledge Atom as a YAML file.
// Returns the relative path from the project root.
func WriteKnowledgeAtom(memoryRoot, branchPackageID string, atom *agent.KnowledgeAtom) (string, error) {
	dir := filepath.Join(memoryRoot, "branches", branchPackageID, "knowledge")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create knowledge directory: %w", err)
	}

	filename := atom.ID + ".yaml"
	filePath := filepath.Join(dir, filename)

	data, err := yaml.Marshal(atom)
	if err != nil {
		return "", fmt.Errorf("failed to marshal atom to YAML: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write atom file: %w", err)
	}

	// Return relative path from project root
	relPath := filepath.Join("prompts", "memory", "branches", branchPackageID, "knowledge", filename)
	return relPath, nil
}
