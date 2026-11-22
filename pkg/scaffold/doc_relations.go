package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	docRelationsPrefix = "doc-relations-"
	docRelationsMaxPer = 40
	farKnowledgePrefix = "__far-knowledge-"
)

// SimilarHit is one similarity candidate from latest/.
type SimilarHit struct {
	LatestRelPath string
	Score         float64
}

// DocRelation maps one backup file to latest hints.
type DocRelation struct {
	BackupRelPath string
	UID           string
	UIDMatchPath  string
	SimilarTop3   []SimilarHit
}

// ParseFrontmatterUID extracts uid from YAML frontmatter between --- fences.
func ParseFrontmatterUID(content []byte) (uid string, ok bool) {
	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return "", false
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	fm := rest[:end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "uid:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "uid:"))
			v = strings.Trim(v, `"'`)
			if v != "" {
				return v, true
			}
		}
	}
	return "", false
}

// ParseFrontmatterID extracts id from YAML frontmatter.
func ParseFrontmatterID(content []byte) string {
	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return ""
	}
	rest := text[3:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return ""
	}
	fm := rest[:end]
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "id:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			v = strings.Trim(v, `"'`)
			return v
		}
	}
	return ""
}

// ShouldSkipRelationPath reports whether a relative path is excluded from relations.
func ShouldSkipRelationPath(rel string) bool {
	base := filepath.Base(rel)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if strings.HasPrefix(name, farKnowledgePrefix) || strings.HasPrefix(name, "__far_knowledge_") {
		return true
	}
	return false
}

type fileContent struct {
	rel  string
	data []byte
}

func listMarkdownFiles(root string) ([]fileContent, error) {
	var out []fileContent
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if ShouldSkipRelationPath(rel) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		id := ParseFrontmatterID(data)
		if strings.HasPrefix(id, farKnowledgePrefix) {
			return nil
		}
		out = append(out, fileContent{rel: rel, data: data})
		return nil
	})
	return out, err
}

// BuildDocRelations builds relations from backupRoot looking toward latestRoot.
func BuildDocRelations(backupRoot, latestRoot string) ([]DocRelation, error) {
	backupFiles, err := listMarkdownFiles(backupRoot)
	if err != nil {
		return nil, err
	}
	latestFiles, err := listMarkdownFiles(latestRoot)
	if err != nil {
		return nil, err
	}

	uidToLatest := make(map[string]string)
	for _, f := range latestFiles {
		if uid, ok := ParseFrontmatterUID(f.data); ok {
			uidToLatest[uid] = f.rel
		}
	}

	sort.Slice(backupFiles, func(i, j int) bool { return backupFiles[i].rel < backupFiles[j].rel })

	var rels []DocRelation
	for _, b := range backupFiles {
		rel := DocRelation{BackupRelPath: b.rel}
		if uid, ok := ParseFrontmatterUID(b.data); ok {
			rel.UID = uid
			if p, hit := uidToLatest[uid]; hit {
				rel.UIDMatchPath = p
			}
		}
		if rel.UIDMatchPath == "" {
			type scored struct {
				rel   string
				score float64
			}
			var scores []scored
			for _, l := range latestFiles {
				scores = append(scores, scored{
					rel:   l.rel,
					score: SimilarityScore(string(b.data), string(l.data), DefaultNGramN),
				})
			}
			sort.Slice(scores, func(i, j int) bool {
				if scores[i].score == scores[j].score {
					return scores[i].rel < scores[j].rel
				}
				return scores[i].score > scores[j].score
			})
			limit := 3
			if len(scores) < limit {
				limit = len(scores)
			}
			for i := 0; i < limit; i++ {
				rel.SimilarTop3 = append(rel.SimilarTop3, SimilarHit{
					LatestRelPath: scores[i].rel,
					Score:         scores[i].score,
				})
			}
		}
		rels = append(rels, rel)
	}
	return rels, nil
}

// WriteDocRelationFiles writes doc-relations-NNN.md under workdir. Always starts at 001.
func WriteDocRelationFiles(workdir string, rels []DocRelation, maxPerFile int) ([]string, error) {
	if maxPerFile <= 0 {
		maxPerFile = docRelationsMaxPer
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return nil, err
	}
	if len(rels) == 0 {
		path := filepath.Join(workdir, fmt.Sprintf("%s001.md", docRelationsPrefix))
		body := "# Document relations\n\nNo backup markdown files to relate.\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, err
		}
		return []string{filepath.Base(path)}, nil
	}

	var written []string
	fileIdx := 1
	for start := 0; start < len(rels); start += maxPerFile {
		end := start + maxPerFile
		if end > len(rels) {
			end = len(rels)
		}
		chunk := rels[start:end]
		name := fmt.Sprintf("%s%03d.md", docRelationsPrefix, fileIdx)
		path := filepath.Join(workdir, name)
		var b strings.Builder
		b.WriteString("# Document relations\n\n")
		b.WriteString(fmt.Sprintf("Part %03d. Paths are relative to backup/ and latest/.\n\n", fileIdx))
		for _, r := range chunk {
			b.WriteString(fmt.Sprintf("## `%s`\n\n", r.BackupRelPath))
			if r.UID != "" {
				b.WriteString(fmt.Sprintf("- uid: `%s`\n", r.UID))
			} else {
				b.WriteString("- uid: _(missing)_\n")
			}
			if r.UIDMatchPath != "" {
				b.WriteString(fmt.Sprintf("- uid match in latest: `%s`\n", r.UIDMatchPath))
			} else {
				b.WriteString("- uid match in latest: _(none)_\n")
				b.WriteString("- similar top 3:\n")
				for _, h := range r.SimilarTop3 {
					b.WriteString(fmt.Sprintf("  - `%s` (score=%.4f)\n", h.LatestRelPath, h.Score))
				}
			}
			b.WriteString("\n")
		}
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			return nil, err
		}
		written = append(written, name)
		fileIdx++
	}
	return written, nil
}
