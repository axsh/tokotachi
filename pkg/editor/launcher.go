package editor

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/axsh/tokotachi/pkg/detect"
)

// CustomLauncher implements Launcher dynamically using the configuration loaded from editor.yaml.
type CustomLauncher struct {
	name string
	cfg  EditorConfig
}

// NewLauncher creates a new Launcher for the given editor based on the Config.
func NewLauncher(ed detect.Editor, cfg *Config) (Launcher, error) {
	editorCfg, ok := cfg.ResolveEditor(string(ed))
	if !ok {
		return nil, fmt.Errorf("editor %q not found in config", ed)
	}
	return &CustomLauncher{
		name: string(ed),
		cfg:  editorCfg,
	}, nil
}

// Name returns the editor identifier.
func (l *CustomLauncher) Name() string { return l.name }

// Launch opens the editor using CustomLauncher configuration.
func (l *CustomLauncher) Launch(opts LaunchOptions) (LaunchResult, error) {
	// 1. Resolve cmd and launchType
	cmd := l.cfg.Cmd
	launchType := l.cfg.Type

	// OS-specific override
	var platCfg *PlatformConfig
	switch runtime.GOOS {
	case "windows":
		platCfg = l.cfg.Windows
	case "darwin":
		platCfg = l.cfg.Darwin
	case "linux":
		platCfg = l.cfg.Linux
	}

	if platCfg != nil {
		if platCfg.Cmd != nil && *platCfg.Cmd != "" {
			cmd = *platCfg.Cmd
		}
		if platCfg.Type != nil && *platCfg.Type != "" {
			launchType = *platCfg.Type
		}
	}

	// Environment variable override (highest priority, backward compatibility)
	envKey := ""
	switch l.name {
	case "code":
		envKey = "TT_CMD_CODE"
	case "cursor":
		envKey = "TT_CMD_CURSOR"
	case "ag":
		envKey = "TT_CMD_AG"
	case "claude":
		envKey = "TT_CMD_CLAUDE"
	}
	if envKey != "" {
		if envCmd := os.Getenv(envKey); envCmd != "" {
			cmd = envCmd
		}
	}

	// 2. Determine if devcontainer attach should be attempted
	isDevcontainer := opts.TryDevcontainer && opts.ContainerName != "" && launchType == "vscode"

	// 3. Resolve argument template
	var argsTmpl []string
	method := "local"

	var platArgs *ArgsConfig
	if platCfg != nil {
		platArgs = platCfg.Args
	}

	if isDevcontainer {
		method = "devcontainer"
		if platArgs != nil && len(platArgs.Devcontainer) > 0 {
			argsTmpl = platArgs.Devcontainer
		} else {
			argsTmpl = l.cfg.Args.Devcontainer
		}
	} else if opts.NewWindow && launchType == "vscode" {
		if platArgs != nil && len(platArgs.NewWindow) > 0 {
			argsTmpl = platArgs.NewWindow
		} else {
			argsTmpl = l.cfg.Args.NewWindow
		}
	} else {
		if platArgs != nil && len(platArgs.Default) > 0 {
			argsTmpl = platArgs.Default
		} else {
			argsTmpl = l.cfg.Args.Default
		}
	}

	if launchType == "cli" {
		method = "cli"
	}

	// 4. Resolve placeholders
	uri := ""
	if isDevcontainer {
		uri = DevcontainerURI(opts.ContainerName, "")
	}

	args := make([]string, len(argsTmpl))
	for i, arg := range argsTmpl {
		replaced := strings.ReplaceAll(arg, "{path}", opts.WorktreePath)
		replaced = strings.ReplaceAll(replaced, "{container}", opts.ContainerName)
		replaced = strings.ReplaceAll(replaced, "{uri}", uri)
		args[i] = replaced
	}

	// 5. Dry-run execution
	if opts.DryRun {
		opts.Logger.Info("[DRY-RUN] %s %s (method: %s)", cmd, strings.Join(args, " "), method)
		return LaunchResult{Method: method, EditorCmd: cmd}, nil
	}

	// 6. Real execution
	if isDevcontainer {
		opts.Logger.Info("Attempting Dev Container attach for %s...", opts.ContainerName)
		if err := opts.CmdRunner.RunInteractive(cmd, args...); err == nil {
			opts.Logger.Info("Dev Container attach succeeded")
			return LaunchResult{Method: "devcontainer", EditorCmd: cmd}, nil
		}
		opts.Logger.Warn("Dev Container attach failed, falling back to local open")

		// Fallback to local open
		fallbackArgsTmpl := l.cfg.Args.Default
		if platArgs != nil && len(platArgs.Default) > 0 {
			fallbackArgsTmpl = platArgs.Default
		}

		fallbackArgs := make([]string, len(fallbackArgsTmpl))
		for i, arg := range fallbackArgsTmpl {
			replaced := strings.ReplaceAll(arg, "{path}", opts.WorktreePath)
			replaced = strings.ReplaceAll(replaced, "{container}", opts.ContainerName)
			replaced = strings.ReplaceAll(replaced, "{uri}", "")
			fallbackArgs[i] = replaced
		}

		if err := opts.CmdRunner.RunInteractive(cmd, fallbackArgs...); err != nil {
			return LaunchResult{}, fmt.Errorf("failed to open editor %s after devcontainer fallback: %w", l.name, err)
		}
		return LaunchResult{Method: "local", Fallback: true, EditorCmd: cmd}, nil
	}

	if err := opts.CmdRunner.RunInteractive(cmd, args...); err != nil {
		return LaunchResult{}, fmt.Errorf("failed to open editor %s: %w", l.name, err)
	}

	return LaunchResult{Method: method, EditorCmd: cmd}, nil
}
