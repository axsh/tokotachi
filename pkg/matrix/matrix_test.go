package matrix_test

import (
	"testing"

	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/pkg/matrix"
	"github.com/stretchr/testify/assert"
)

func TestResolveCapability(t *testing.T) {
	tests := []struct {
		name                 string
		os                   detect.OS
		editor               detect.Editor
		wantLocalOpen        matrix.CompatLevel
		wantDevcontainerOpen matrix.CompatLevel
		wantSSH              matrix.CompatLevel
	}{
		{
			name:                 "linux+vscode",
			os:                   detect.OSLinux,
			editor:               detect.EditorVSCode,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L2BestEffort,
			wantSSH:              matrix.L1Supported,
		},
		{
			name:                 "linux+cursor",
			os:                   detect.OSLinux,
			editor:               detect.EditorCursor,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L2BestEffort,
			wantSSH:              matrix.L1Supported,
		},
		{
			name:                 "linux+ag",
			os:                   detect.OSLinux,
			editor:               detect.EditorAG,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L4Unsupported,
			wantSSH:              matrix.L2BestEffort,
		},
		{
			name:                 "linux+claude",
			os:                   detect.OSLinux,
			editor:               detect.EditorClaude,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L4Unsupported,
			wantSSH:              matrix.L1Supported,
		},
		{
			name:                 "macos+vscode",
			os:                   detect.OSMacOS,
			editor:               detect.EditorVSCode,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L2BestEffort,
			wantSSH:              matrix.L1Supported,
		},
		{
			name:                 "macos+ag",
			os:                   detect.OSMacOS,
			editor:               detect.EditorAG,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L4Unsupported,
			wantSSH:              matrix.L2BestEffort,
		},
		{
			name:                 "windows+vscode",
			os:                   detect.OSWindows,
			editor:               detect.EditorVSCode,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L2BestEffort,
			wantSSH:              matrix.L2BestEffort,
		},
		{
			name:                 "windows+claude",
			os:                   detect.OSWindows,
			editor:               detect.EditorClaude,
			wantLocalOpen:        matrix.L1Supported,
			wantDevcontainerOpen: matrix.L1Supported,
			wantSSH:              matrix.L2BestEffort,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cap := matrix.ResolveCapability(tt.os, tt.editor)
			assert.Equal(t, tt.wantLocalOpen, cap.LocalOpenLevel)
			assert.Equal(t, tt.wantDevcontainerOpen, cap.DevcontainerOpenLevel)
			assert.Equal(t, tt.wantSSH, cap.SSHLevel)
		})
	}
}

func TestResolveCapability_AllCombinations(t *testing.T) {
	oses := []detect.OS{detect.OSLinux, detect.OSMacOS, detect.OSWindows}
	editors := []detect.Editor{detect.EditorVSCode, detect.EditorCursor, detect.EditorAG, detect.EditorClaude}

	for _, os := range oses {
		for _, editor := range editors {
			t.Run(string(os)+"+"+string(editor), func(t *testing.T) {
				cap := matrix.ResolveCapability(os, editor)
				assert.True(t, cap.CanOpenLocal, "CanOpenLocal should always be true")
			})
		}
	}
}
