package detect_test

import (
	"testing"

	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEditor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    detect.Editor
		wantErr bool
	}{
		{"code", "code", detect.EditorVSCode, false},
		{"cursor", "cursor", detect.EditorCursor, false},
		{"ag", "ag", detect.EditorAG, false},
		{"claude", "claude", detect.EditorClaude, false},
		{"vscode alias", "vscode", detect.EditorVSCode, false},
		{"antigravity alias", "antigravity", detect.EditorAG, false},
		{"invalid", "vim", detect.Editor(""), true},
		{"empty", "", detect.Editor(""), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detect.ParseEditor(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestResolveEditor(t *testing.T) {
	// Resolution priority: CLI flag > env var > feature config > global config > default
	tests := []struct {
		name          string
		cliFlag       string
		envVar        string
		featureConfig string
		globalConfig  string
		want          detect.Editor
	}{
		{
			name:    "cli flag takes highest priority",
			cliFlag: "code",
			envVar:  "ag",
			want:    detect.EditorVSCode,
		},
		{
			name:         "env var overrides config",
			envVar:       "ag",
			globalConfig: "cursor",
			want:         detect.EditorAG,
		},
		{
			name:          "feature config overrides global",
			featureConfig: "claude",
			globalConfig:  "cursor",
			want:          detect.EditorClaude,
		},
		{
			name:         "global config used as fallback",
			globalConfig: "code",
			want:         detect.EditorVSCode,
		},
		{
			name: "default is cursor when nothing set",
			want: detect.EditorCursor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detect.ResolveEditor(tt.cliFlag, tt.envVar, tt.featureConfig, tt.globalConfig)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveEditor_InvalidValue(t *testing.T) {
	_, err := detect.ResolveEditor("vim", "", "", "")
	require.Error(t, err)
}
