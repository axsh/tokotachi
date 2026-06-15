package knowledge

import "time"

// CategoryMeta represents _category.yaml content.
type CategoryMeta struct {
	CategoryID  string    `yaml:"category_id"`
	Title       string    `yaml:"title"`
	Description string    `yaml:"description,omitempty"`
	CreatedAt   time.Time `yaml:"created_at"`
	LastUpdated time.Time `yaml:"last_updated"`
}

// KnowledgeFileMeta represents frontmatter of a knowledge .md file.
type KnowledgeFileMeta struct {
	KnowledgeID    string    `yaml:"knowledge_id"`
	Title          string    `yaml:"title"`
	CategoryPath   string    `yaml:"category_path"`
	CreatedAt      time.Time `yaml:"created_at"`
	LastUpdated    time.Time `yaml:"last_updated"`
	SourceEventIDs []string  `yaml:"source_event_ids"`
}

// CategoryInfo holds category tree statistics.
type CategoryInfo struct {
	Path         string `json:"path"`
	Title        string `json:"title"`
	FileCount    int    `json:"file_count"`
	TotalSize    int64  `json:"total_size"`
	LastUpdated  string `json:"last_updated"`
	HasSubdirs   bool   `json:"has_subdirs"`
}

// SplitPlan represents a split operation plan.
type SplitPlan struct {
	Assignments map[string]string `json:"assignments"` // knowledge_file -> target_subcategory
}

// MergePlan represents a merge operation plan.
type MergePlan struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
