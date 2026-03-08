package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectLocale_FromLANG(t *testing.T) {
	t.Setenv("LANG", "ja_JP.UTF-8")
	t.Setenv("LC_ALL", "")

	locale := DetectLocale("")
	assert.Equal(t, "ja", locale)
}

func TestDetectLocale_FromLCAll(t *testing.T) {
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("LANG", "ja_JP.UTF-8")

	locale := DetectLocale("")
	assert.Equal(t, "en", locale)
}

func TestDetectLocale_ExplicitFlag(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")

	locale := DetectLocale("ja")
	assert.Equal(t, "ja", locale)
}

func TestDetectLocale_Empty(t *testing.T) {
	t.Setenv("LANG", "")
	t.Setenv("LC_ALL", "")

	locale := DetectLocale("")
	assert.Equal(t, "", locale)
}

func TestDetectLocale_CLocale(t *testing.T) {
	t.Setenv("LANG", "C")
	t.Setenv("LC_ALL", "")

	locale := DetectLocale("")
	assert.Equal(t, "", locale)
}

func TestMergeLocaleFiles_WithOverlay(t *testing.T) {
	base := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("English README")},
		{RelativePath: "scripts/.gitkeep", Content: []byte("")},
	}
	locale := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("日本語 README")},
	}

	merged := MergeLocaleFiles(base, locale)

	assert.Len(t, merged, 2)
	fileMap := make(map[string]string)
	for _, f := range merged {
		fileMap[f.RelativePath] = string(f.Content)
	}
	assert.Equal(t, "日本語 README", fileMap["README.md"])
	assert.Equal(t, "", fileMap["scripts/.gitkeep"])
}

func TestMergeLocaleFiles_NoOverlay(t *testing.T) {
	base := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("English README")},
	}

	merged := MergeLocaleFiles(base, nil)
	assert.Len(t, merged, 1)
	assert.Equal(t, "English README", string(merged[0].Content))
}

func TestMergeLocaleFiles_PartialOverlay(t *testing.T) {
	base := []DownloadedFile{
		{RelativePath: "features/README.md", Content: []byte("English features")},
		{RelativePath: "work/README.md", Content: []byte("English work")},
		{RelativePath: "scripts/.gitkeep", Content: []byte("")},
	}
	locale := []DownloadedFile{
		{RelativePath: "features/README.md", Content: []byte("日本語 features")},
	}

	merged := MergeLocaleFiles(base, locale)
	assert.Len(t, merged, 3)

	fileMap := make(map[string]string)
	for _, f := range merged {
		fileMap[f.RelativePath] = string(f.Content)
	}
	assert.Equal(t, "日本語 features", fileMap["features/README.md"])
	assert.Equal(t, "English work", fileMap["work/README.md"])
}

func TestMergeLocaleFiles_LocaleOnlyFiles(t *testing.T) {
	base := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("English")},
	}
	locale := []DownloadedFile{
		{RelativePath: "README.md", Content: []byte("日本語")},
		{RelativePath: "GUIDE.md", Content: []byte("日本語ガイド")},
	}

	merged := MergeLocaleFiles(base, locale)
	assert.Len(t, merged, 2)

	fileMap := make(map[string]string)
	for _, f := range merged {
		fileMap[f.RelativePath] = string(f.Content)
	}
	assert.Equal(t, "日本語", fileMap["README.md"])
	assert.Equal(t, "日本語ガイド", fileMap["GUIDE.md"])
}
