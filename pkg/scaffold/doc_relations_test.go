package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatterUID(t *testing.T) {
	uid, ok := ParseFrontmatterUID([]byte("---\nuid: 11111111-1111-1111-1111-111111111111\nid: x\n---\n\n# Hi\n"))
	require.True(t, ok)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", uid)
}

func TestWriteDocRelationFiles_AlwaysNumbered(t *testing.T) {
	dir := t.TempDir()
	names, err := WriteDocRelationFiles(dir, nil, 10)
	require.NoError(t, err)
	require.Equal(t, []string{"doc-relations-001.md"}, names)
	_, err = os.Stat(filepath.Join(dir, "doc-relations.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestBuildDocRelations_UIDAndSimilar(t *testing.T) {
	root := t.TempDir()
	backup := filepath.Join(root, "backup")
	latest := filepath.Join(root, "latest")
	require.NoError(t, os.MkdirAll(filepath.Join(backup, "capabilities"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(latest, "capabilities"), 0o755))

	uid := "22222222-2222-2222-2222-222222222222"
	oldPath := filepath.Join(backup, "capabilities", "old-name.md")
	newPath := filepath.Join(latest, "capabilities", "new-name.md")
	body := "---\napiVersion: agent.meta/v1\nkind: capability\nid: renamed\nuid: " + uid + "\ntitle: T\ndescription: d\nbody: inline\ntags:\n  - baseline\n---\n\n# Body shared across rename\n"
	require.NoError(t, os.WriteFile(oldPath, []byte(body), 0o644))
	require.NoError(t, os.WriteFile(newPath, []byte(body), 0o644))

	noUIDBackup := filepath.Join(backup, "capabilities", "local-only.md")
	require.NoError(t, os.WriteFile(noUIDBackup, []byte("---\nid: local-only\ntitle: L\ndescription: d\nbody: inline\n---\n\n# Local only unique text xyzzy\n"), 0o644))
	latestOther := filepath.Join(latest, "capabilities", "other.md")
	require.NoError(t, os.WriteFile(latestOther, []byte("---\nid: other\ntitle: O\ndescription: d\nbody: inline\n---\n\n# Local only unique text xyzzy with more\n"), 0o644))

	far := filepath.Join(backup, "capabilities", "__far-knowledge-skip.md")
	require.NoError(t, os.WriteFile(far, []byte("---\nid: __far-knowledge-skip\n---\n\n# skip\n"), 0o644))

	rels, err := BuildDocRelations(backup, latest)
	require.NoError(t, err)
	var foundUID, foundSimilar, foundFar bool
	for _, r := range rels {
		if strings.Contains(r.BackupRelPath, "old-name") {
			foundUID = true
			assert.Equal(t, "capabilities/new-name.md", r.UIDMatchPath)
		}
		if strings.Contains(r.BackupRelPath, "local-only") {
			foundSimilar = true
			require.NotEmpty(t, r.SimilarTop3)
		}
		if strings.Contains(r.BackupRelPath, "__far-knowledge") {
			foundFar = true
		}
	}
	assert.True(t, foundUID)
	assert.True(t, foundSimilar)
	assert.False(t, foundFar)
}

func TestIsCleanUpdate(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	require.NoError(t, os.MkdirAll(a, 0o755))
	require.NoError(t, os.MkdirAll(b, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(a, "f.md"), []byte("same"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(b, "f.md"), []byte("same"), 0o644))
	assert.True(t, IsCleanUpdate(a, b))
	require.NoError(t, os.WriteFile(filepath.Join(a, "f.md"), []byte("diff"), 0o644))
	assert.False(t, IsCleanUpdate(a, b))
}
