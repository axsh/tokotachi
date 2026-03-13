package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShardPath(t *testing.T) {
	tests := []struct {
		category string
		name     string
		expected string
	}{
		{"root", "default", "catalog/scaffolds/6/j/v/n.yaml"},
		{"feature", "axsh-go-standard", "catalog/scaffolds/b/i/b/l.yaml"},
		{"project", "axsh-go-standard", "catalog/scaffolds/8/w/4/o.yaml"},
		{"feature", "axsh-go-kotoshiro-mcp", "catalog/scaffolds/i/4/2/h.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.category+"/"+tt.name, func(t *testing.T) {
			got := ShardPath(tt.category, tt.name)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestShardPath_Constants(t *testing.T) {
	assert.Equal(t, "root", DefaultCategory)
	assert.Equal(t, "default", DefaultName)
	assert.Contains(t, KnownCategories, "root")
	assert.Contains(t, KnownCategories, "project")
	assert.Contains(t, KnownCategories, "feature")
}
