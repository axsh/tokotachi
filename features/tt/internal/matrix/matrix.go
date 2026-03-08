package matrix

import "github.com/axsh/tokotachi/features/devctl/internal/detect"

// ResolveCapability returns the Capability for the given OS and Editor.
// This is the central matrix lookup function.
func ResolveCapability(os detect.OS, editor detect.Editor) Capability {
	key := matrixKey{os: os, editor: editor}
	if cap, ok := defaultMatrix[key]; ok {
		return cap
	}
	// Fallback: local open only
	return Capability{
		CanOpenLocal:          true,
		LocalOpenLevel:        L1Supported,
		DevcontainerOpenLevel: L4Unsupported,
		SSHLevel:              L4Unsupported,
	}
}

type matrixKey struct {
	os     detect.OS
	editor detect.Editor
}

// defaultMatrix encodes the specification's OS×Editor compatibility table.
// All 12 combinations (3 OS × 4 Editor) are listed.
var defaultMatrix = map[matrixKey]Capability{
	// --- Linux ---
	{detect.OSLinux, detect.EditorVSCode}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
		CanLaunchNewWindow: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
	},
	{detect.OSLinux, detect.EditorCursor}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
		CanLaunchNewWindow: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
	},
	{detect.OSLinux, detect.EditorAG}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: false, CanUseSSHMode: true,
		RequiresBestEffort: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L2BestEffort,
	},
	{detect.OSLinux, detect.EditorClaude}: {
		CanOpenLocal: true, CanRunClaudeLocally: true, CanUseSSHMode: true,
		LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L1Supported,
	},
	// --- macOS ---
	{detect.OSMacOS, detect.EditorVSCode}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
		CanLaunchNewWindow: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
	},
	{detect.OSMacOS, detect.EditorCursor}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
		CanLaunchNewWindow: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L1Supported,
	},
	{detect.OSMacOS, detect.EditorAG}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: false, CanUseSSHMode: true,
		RequiresBestEffort: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L2BestEffort,
	},
	{detect.OSMacOS, detect.EditorClaude}: {
		CanOpenLocal: true, CanRunClaudeLocally: true, CanUseSSHMode: true,
		LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L1Supported,
	},
	// --- Windows ---
	{detect.OSWindows, detect.EditorVSCode}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
		RequiresBestEffort: true, CanLaunchNewWindow: true,
		LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L2BestEffort,
	},
	{detect.OSWindows, detect.EditorCursor}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: true, CanUseSSHMode: true,
		RequiresBestEffort: true, CanLaunchNewWindow: true,
		LocalOpenLevel: L1Supported, DevcontainerOpenLevel: L2BestEffort, SSHLevel: L2BestEffort,
	},
	{detect.OSWindows, detect.EditorAG}: {
		CanOpenLocal: true, CanTryDevcontainerAttach: false, CanUseSSHMode: true,
		RequiresBestEffort: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L4Unsupported, SSHLevel: L2BestEffort,
	},
	{detect.OSWindows, detect.EditorClaude}: {
		CanOpenLocal: true, CanRunClaudeLocally: true, CanUseSSHMode: true,
		RequiresBestEffort: true,
		LocalOpenLevel:     L1Supported, DevcontainerOpenLevel: L1Supported, SSHLevel: L2BestEffort,
	},
}
