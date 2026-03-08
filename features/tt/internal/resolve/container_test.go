package resolve_test

import (
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/stretchr/testify/assert"
)

func TestContainerName(t *testing.T) {
	tests := []struct {
		project string
		feature string
		want    string
	}{
		{"myproj", "feature-a", "myproj-feature-a"},
		{"myproj", "feature_b", "myproj-feature-b"},
		{"myproj", "Feature.C", "myproj-feature-c"},
		{"myproj", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := resolve.ContainerName(tt.project, tt.feature)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestImageName(t *testing.T) {
	got := resolve.ImageName("myproj", "feature-a")
	assert.Equal(t, "myproj-dev-feature-a", got)
}

func TestImageName_EmptyFeature(t *testing.T) {
	got := resolve.ImageName("myproj", "")
	assert.Equal(t, "", got)
}
