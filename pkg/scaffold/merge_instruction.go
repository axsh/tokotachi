package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DefaultMergeInstruction returns the English AI merge instruction for --update.
func DefaultMergeInstruction(workdir string, relationFiles []string) string {
	var b strings.Builder
	b.WriteString("# Scaffold manifest update — merge instructions\n\n")
	b.WriteString("You are merging an upstream scaffold manifest into this project's `prompts/manifest/`.\n\n")
	b.WriteString(fmt.Sprintf("Work directory: `%s`\n\n", filepath.ToSlash(workdir)))
	b.WriteString("## Layout\n\n")
	b.WriteString(fmt.Sprintf("- `%s/backup/` — snapshot of the local `prompts/manifest/` at update start\n", filepath.ToSlash(workdir)))
	b.WriteString(fmt.Sprintf("- `%s/latest/` — freshly downloaded upstream manifest content\n", filepath.ToSlash(workdir)))
	b.WriteString("- Relation hints:\n")
	for _, f := range relationFiles {
		b.WriteString(fmt.Sprintf("  - `%s`\n", filepath.ToSlash(filepath.Join(workdir, f))))
	}
	b.WriteString("\n## Merge policies (mandatory)\n\n")
	b.WriteString("1. Do not discard or rewrite project-specific information without cause.\n")
	b.WriteString("2. Prefer incorporating improvements and convenience from `latest/`.\n")
	b.WriteString("3. Do not break existing references (`uses_capabilities`, template refs, relative links, skill ids).\n")
	b.WriteString("4. Keep current behavior working while absorbing upstream improvements.\n")
	b.WriteString("5. Use `doc-relations-NNN.md` (zero-padded sequence starting at 001, even when only one file) as mechanical hints; do not invent renames that contradict uid matches.\n")
	b.WriteString("6. After merging into the real `prompts/manifest/`, leave the workdir intact for review unless the user asks to delete it.\n")
	b.WriteString("\n## Procedure\n\n")
	b.WriteString("1. Read every listed `doc-relations-NNN.md` file.\n")
	b.WriteString("2. For uid matches, treat backup and latest paths as the same document even if renamed.\n")
	b.WriteString("3. For missing uids, use similar top-3 scores as hints only — never as sole authority.\n")
	b.WriteString("4. Apply the merged result to the project's `prompts/manifest/` (not only inside the workdir).\n")
	b.WriteString("5. Preserve local-only skills and project-specific wording unless upstream clearly supersedes them.\n")
	return b.String()
}
