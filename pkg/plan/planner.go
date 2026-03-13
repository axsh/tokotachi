package plan

import (
	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/pkg/matrix"
)

// Input represents the user's request and resolved environment.
type Input struct {
	Feature       string
	OS            detect.OS
	Editor        detect.Editor
	ContainerMode matrix.ContainerMode
	Up            bool
	Down          bool
	Status        bool
	Shell         bool
	Exec          []string
	SSH           bool
	Rebuild       bool
	NoBuild       bool
	// EditorOpen indicates the editor should be opened (set when --editor on up, or open subcommand).
	EditorOpen bool
	// Attach indicates DevContainer attach should be attempted (open --attach).
	Attach bool
}

// Plan describes the concrete actions to execute.
type Plan struct {
	ShouldStartContainer  bool
	ShouldStopContainer   bool
	ShouldOpenEditor      bool
	ShouldShowStatus      bool
	ShouldOpenShell       bool
	ExecCommand           []string
	TryDevcontainerAttach bool
	SSHMode               bool
	Rebuild               bool
	NoBuild               bool
	CompatLevel           matrix.CompatLevel
}

// Build constructs a Plan from the Input by consulting the matrix.
func Build(input Input) Plan {
	cap := matrix.ResolveCapability(input.OS, input.Editor)

	p := Plan{
		ShouldStartContainer: input.Up,
		ShouldStopContainer:  input.Down,
		ShouldShowStatus:     input.Status,
		ShouldOpenShell:      input.Shell,
		ExecCommand:          input.Exec,
		SSHMode:              input.SSH,
		Rebuild:              input.Rebuild,
		NoBuild:              input.NoBuild,
	}

	if input.EditorOpen {
		p.ShouldOpenEditor = true
		// Determine whether to attempt devcontainer attach
		if input.Attach && cap.CanTryDevcontainerAttach &&
			(input.ContainerMode == matrix.ContainerDevContainer ||
				input.ContainerMode == matrix.ContainerDockerLocal) {
			p.TryDevcontainerAttach = true
			p.CompatLevel = cap.DevcontainerOpenLevel
		} else if input.Up && cap.CanTryDevcontainerAttach &&
			(input.ContainerMode == matrix.ContainerDevContainer ||
				input.ContainerMode == matrix.ContainerDockerLocal) {
			// up --editor also tries devcontainer attach
			p.TryDevcontainerAttach = true
			p.CompatLevel = cap.DevcontainerOpenLevel
		} else {
			p.CompatLevel = cap.LocalOpenLevel
		}
	}

	return p
}
