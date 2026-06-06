package scaffold

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveRepoURL(t *testing.T) {
	tests := []struct {
		name         string
		specifiedURL string
		envURL       string
		expectedURL  string
	}{
		{
			name:         "Default fallback when no flag and no env",
			specifiedURL: "",
			envURL:       "",
			expectedURL:  "https://github.com/axsh/tokotachi",
		},
		{
			name:         "Env override when no flag",
			specifiedURL: "",
			envURL:       "https://github.com/some-owner/some-repo",
			expectedURL:  "https://github.com/some-owner/some-repo",
		},
		{
			name:         "Flag takes precedence over env",
			specifiedURL: "https://github.com/flag-owner/flag-repo",
			envURL:       "https://github.com/some-owner/some-repo",
			expectedURL:  "https://github.com/flag-owner/flag-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envURL != "" {
				t.Setenv("TT_CONTENT_REPO", tt.envURL)
			} else {
				os.Unsetenv("TT_CONTENT_REPO")
			}
			actual := resolveRepoURL(tt.specifiedURL)
			assert.Equal(t, tt.expectedURL, actual)
		})
	}
}
