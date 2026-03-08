package matrix

import "github.com/axsh/tokotachi/features/devctl/internal/detect"

// ContainerMode represents how containers are used.
type ContainerMode string

const (
	ContainerNone         ContainerMode = "none"
	ContainerDevContainer ContainerMode = "devcontainer"
	ContainerDockerLocal  ContainerMode = "docker-local"
	ContainerDockerSSH    ContainerMode = "docker-ssh"
)

// Action represents the user-requested operation.
type Action string

const (
	ActionUp     Action = "up"
	ActionOpen   Action = "open"
	ActionUpOpen Action = "up_open"
	ActionDown   Action = "down"
	ActionShell  Action = "shell"
	ActionExec   Action = "exec"
	ActionStatus Action = "status"
)

// CompatLevel represents the support level for a combination.
type CompatLevel int

const (
	L1Supported   CompatLevel = iota // Full support
	L2BestEffort                     // Try, fallback on failure
	L3Fallback                       // Direct fallback
	L4Unsupported                    // Error or no-op
)

// String returns the human-readable level name.
func (l CompatLevel) String() string {
	switch l {
	case L1Supported:
		return "L1:Supported"
	case L2BestEffort:
		return "L2:BestEffort"
	case L3Fallback:
		return "L3:Fallback"
	case L4Unsupported:
		return "L4:Unsupported"
	default:
		return "Unknown"
	}
}

// Context holds the resolved environment for matrix lookup.
type Context struct {
	OS            detect.OS
	Editor        detect.Editor
	ContainerMode ContainerMode
	Action        Action
}
