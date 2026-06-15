package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store manages the knowledge directory tree under rootDir.
type Store struct {
	RootDir string // e.g., prompts/memory/knowledge/
}

// NewStore creates a new Store.
func NewStore(rootDir string) *Store {
	return &Store{RootDir: rootDir}
}

// Add creates a new category directory (if needed) with _category.yaml
// and writes a knowledge file.
func (s *Store) Add(categoryPath, title, description, contentFile string, sourceEvents []string) error {
	catDir := filepath.Join(s.RootDir, categoryPath)
	catID := filepath.Base(categoryPath)

	// Create category directory
	if err := os.MkdirAll(catDir, 0o755); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	// Create _category.yaml if it doesn't exist
	metaPath := filepath.Join(catDir, categoryMetaFile)
	now := time.Now().UTC()
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		catMeta := &CategoryMeta{
			CategoryID:  catID,
			Title:       title, // Use knowledge title as initial category title if description is empty
			Description: description,
			CreatedAt:   now,
			LastUpdated: now,
		}
		if err := WriteCategoryMeta(catDir, catMeta); err != nil {
			return err
		}
	} else {
		// Update last_updated on existing category
		catMeta, err := ReadCategoryMeta(catDir)
		if err != nil {
			return err
		}
		catMeta.LastUpdated = now
		if err := WriteCategoryMeta(catDir, catMeta); err != nil {
			return err
		}
	}

	// Read content
	content, err := os.ReadFile(contentFile)
	if err != nil {
		return fmt.Errorf("failed to read content file: %w", err)
	}

	// Derive knowledge ID from title
	knowledgeID := slugify(title)
	mdPath := filepath.Join(catDir, knowledgeID+".md")

	meta := &KnowledgeFileMeta{
		ID:             knowledgeID,
		KnowledgeID:    knowledgeID,
		Title:          title,
		Status:         "current",
		CategoryPath:   categoryPath,
		CreatedAt:      now,
		LastUpdated:    now,
		SourceEventIDs: sourceEvents,
	}

	return WriteFrontmatter(mdPath, meta, string(content))
}

// Append adds a new knowledge file to an existing category.
func (s *Store) Append(categoryPath, title, contentFile string, sourceEvents []string) error {
	catDir := filepath.Join(s.RootDir, categoryPath)

	// Verify category exists
	if _, err := os.Stat(filepath.Join(catDir, categoryMetaFile)); err != nil {
		return fmt.Errorf("category %s does not exist: %w", categoryPath, err)
	}

	// Read content
	content, err := os.ReadFile(contentFile)
	if err != nil {
		return fmt.Errorf("failed to read content file: %w", err)
	}

	now := time.Now().UTC()

	// Derive knowledge ID from title
	knowledgeID := slugify(title)
	mdPath := filepath.Join(catDir, knowledgeID+".md")

	meta := &KnowledgeFileMeta{
		ID:             knowledgeID,
		KnowledgeID:    knowledgeID,
		Title:          title,
		Status:         "current",
		CategoryPath:   categoryPath,
		CreatedAt:      now,
		LastUpdated:    now,
		SourceEventIDs: sourceEvents,
	}

	if err := WriteFrontmatter(mdPath, meta, string(content)); err != nil {
		return err
	}

	// Update category last_updated
	catMeta, err := ReadCategoryMeta(catDir)
	if err != nil {
		return err
	}
	catMeta.LastUpdated = now
	return WriteCategoryMeta(catDir, catMeta)
}

// List returns the category tree with statistics.
func (s *Store) List() ([]CategoryInfo, error) {
	var result []CategoryInfo

	if _, err := os.Stat(s.RootDir); os.IsNotExist(err) {
		return result, nil
	}

	entries, err := os.ReadDir(s.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read knowledge root: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		catDir := filepath.Join(s.RootDir, entry.Name())
		info, err := s.gatherCategoryInfo(catDir, entry.Name())
		if err != nil {
			continue // skip broken categories
		}
		result = append(result, *info)
	}

	return result, nil
}

func (s *Store) gatherCategoryInfo(catDir, catPath string) (*CategoryInfo, error) {
	meta, err := ReadCategoryMeta(catDir)
	if err != nil {
		return nil, err
	}

	var fileCount int
	var totalSize int64
	var hasSubdirs bool

	entries, err := os.ReadDir(catDir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			hasSubdirs = true
			continue
		}
		if e.Name() == categoryMetaFile {
			continue
		}
		if strings.HasSuffix(e.Name(), ".md") {
			fileCount++
			info, err := e.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
	}

	return &CategoryInfo{
		Path:        catPath,
		Title:       meta.Title,
		FileCount:   fileCount,
		TotalSize:   totalSize,
		LastUpdated: meta.LastUpdated.Format(time.RFC3339),
		HasSubdirs:  hasSubdirs,
	}, nil
}

// Split splits a category into subcategories based on a plan file.
func (s *Store) Split(categoryPath string, intoNames []string, planFile string) error {
	plan, err := ParseSplitPlan(planFile)
	if err != nil {
		return err
	}

	catDir := filepath.Join(s.RootDir, categoryPath)

	// Verify category exists
	catMeta, err := ReadCategoryMeta(catDir)
	if err != nil {
		return fmt.Errorf("category %s does not exist: %w", categoryPath, err)
	}

	now := time.Now().UTC()

	// Create subcategory directories
	for _, name := range intoNames {
		subDir := filepath.Join(catDir, name)
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			return fmt.Errorf("failed to create subcategory %s: %w", name, err)
		}
		subMeta := &CategoryMeta{
			CategoryID:  name,
			Title:       name, // LLM should provide better titles via plan
			Description: fmt.Sprintf("Split from %s", catMeta.Title),
			CreatedAt:   now,
			LastUpdated: now,
		}
		if err := WriteCategoryMeta(subDir, subMeta); err != nil {
			return err
		}
	}

	// Move files according to plan
	for filename, target := range plan.Assignments {
		srcPath := filepath.Join(catDir, filename)
		dstDir := filepath.Join(catDir, target)
		dstPath := filepath.Join(dstDir, filename)

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // skip missing files
		}

		// Update frontmatter category_path
		meta, body, err := ReadFrontmatter(srcPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filename, err)
		}
		meta.CategoryPath = filepath.Join(categoryPath, target)
		meta.LastUpdated = now

		if err := WriteFrontmatter(dstPath, meta, body); err != nil {
			return err
		}
		_ = os.Remove(srcPath)
	}

	return nil
}

// Merge merges multiple categories into one.
func (s *Store) Merge(categoryPaths []string, into string, planFile string) error {
	plan, err := ParseMergePlan(planFile)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	intoDir := filepath.Join(s.RootDir, into)

	// Create target category directory
	if err := os.MkdirAll(intoDir, 0o755); err != nil {
		return fmt.Errorf("failed to create target category: %w", err)
	}

	// Write category metadata
	catID := filepath.Base(into)
	catMeta := &CategoryMeta{
		CategoryID:  catID,
		Title:       plan.Title,
		Description: plan.Description,
		CreatedAt:   now,
		LastUpdated: now,
	}
	if err := WriteCategoryMeta(intoDir, catMeta); err != nil {
		return err
	}

	// Move all knowledge files from source categories
	for _, catPath := range categoryPaths {
		srcDir := filepath.Join(s.RootDir, catPath)
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() || e.Name() == categoryMetaFile {
				continue
			}
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}

			srcPath := filepath.Join(srcDir, e.Name())
			dstPath := filepath.Join(intoDir, e.Name())

			// Update frontmatter
			meta, body, err := ReadFrontmatter(srcPath)
			if err != nil {
				// Copy as-is if frontmatter is unreadable
				data, _ := os.ReadFile(srcPath)
				_ = os.WriteFile(dstPath, data, 0o644)
			} else {
				meta.CategoryPath = into
				meta.LastUpdated = now
				_ = WriteFrontmatter(dstPath, meta, body)
			}
		}

		// Remove source category directory
		_ = os.RemoveAll(srcDir)
	}

	return nil
}

// Rename renames a category directory and updates metadata.
func (s *Store) Rename(oldPath, newPath, newTitle string) error {
	oldDir := filepath.Join(s.RootDir, oldPath)
	newDir := filepath.Join(s.RootDir, newPath)

	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return fmt.Errorf("category %s does not exist", oldPath)
	}

	// Rename directory
	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("failed to rename category: %w", err)
	}

	now := time.Now().UTC()

	// Update _category.yaml
	catMeta, err := ReadCategoryMeta(newDir)
	if err != nil {
		return err
	}
	catMeta.CategoryID = filepath.Base(newPath)
	catMeta.Title = newTitle
	catMeta.LastUpdated = now
	if err := WriteCategoryMeta(newDir, catMeta); err != nil {
		return err
	}

	// Update frontmatter in all knowledge files
	entries, err := os.ReadDir(newDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == categoryMetaFile || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		mdPath := filepath.Join(newDir, e.Name())
		meta, body, err := ReadFrontmatter(mdPath)
		if err != nil {
			continue
		}
		meta.CategoryPath = newPath
		meta.LastUpdated = now
		_ = WriteFrontmatter(mdPath, meta, body)
	}

	return nil
}

// Move moves a knowledge file to a different category.
func (s *Store) Move(fromFile, toCategoryPath string) error {
	srcPath := filepath.Join(s.RootDir, fromFile)
	dstDir := filepath.Join(s.RootDir, toCategoryPath)

	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source file %s does not exist", fromFile)
	}
	if _, err := os.Stat(filepath.Join(dstDir, categoryMetaFile)); os.IsNotExist(err) {
		return fmt.Errorf("target category %s does not exist", toCategoryPath)
	}

	filename := filepath.Base(srcPath)
	dstPath := filepath.Join(dstDir, filename)
	now := time.Now().UTC()

	// Update frontmatter
	meta, body, err := ReadFrontmatter(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}
	meta.CategoryPath = toCategoryPath
	meta.LastUpdated = now

	if err := WriteFrontmatter(dstPath, meta, body); err != nil {
		return err
	}

	// Remove source file
	if err := os.Remove(srcPath); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}

// slugify converts a title to a kebab-case slug for use as filename.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '_' || r == '/' || r == '-' {
			return '-'
		}
		return -1
	}, s)
	// Collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}
