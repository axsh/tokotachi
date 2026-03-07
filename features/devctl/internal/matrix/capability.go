package matrix

// Capability describes what features are available for a given combination.
type Capability struct {
	CanOpenLocal             bool
	CanTryDevcontainerAttach bool
	CanUseSSHMode            bool
	CanLaunchNewWindow       bool
	CanRunClaudeLocally      bool
	CanRunClaudeInContainer  bool
	RequiresBestEffort       bool
	LocalOpenLevel           CompatLevel
	DevcontainerOpenLevel    CompatLevel
	SSHLevel                 CompatLevel
}
