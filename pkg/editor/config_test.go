package editor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/pkg/detect"
	"github.com/axsh/tokotachi/pkg/matrix"
)

func TestLoadConfig_AutoGenerate(t *testing.T) {
	// Setup temporary home directory
	tmpHome, err := ioutil.TempDir("", "tokotachi-home-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Save original home env
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome) // for Windows support

	// Execute load which should auto-generate
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify file existence
	configPath := filepath.Join(tmpHome, ".kotoshiro", "tokotachi", "editor.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected editor.yaml to be auto-generated at %s", configPath)
	}

	// Verify defaults
	if _, ok := cfg.ResolveEditor("code"); !ok {
		t.Error("expected 'code' to be defined in default configuration")
	}
	if _, ok := cfg.ResolveEditor("ag"); !ok {
		t.Error("expected 'ag' to be defined in default configuration")
	}
}

func TestResolveEditor_UserOverride(t *testing.T) {
	cfg := defaultConfig()

	// Initially, code cmd should be "code"
	ec, ok := cfg.ResolveEditor("code")
	if !ok {
		t.Fatal("expected 'code' to exist")
	}
	if ec.Cmd != "code" {
		t.Errorf("expected default cmd 'code', got %q", ec.Cmd)
	}

	// Override in User section
	cfg.User.Editors = map[string]EditorConfig{
		"code": {
			Cmd:  "my-custom-code",
			Type: "vscode",
			Args: ArgsConfig{
				Default: []string{"-w", "{path}"},
			},
		},
	}

	// Resolve again and check override
	ec, ok = cfg.ResolveEditor("code")
	if !ok {
		t.Fatal("expected 'code' to exist after override")
	}
	if ec.Cmd != "my-custom-code" {
		t.Errorf("expected overridden cmd 'my-custom-code', got %q", ec.Cmd)
	}
	if len(ec.Args.Default) != 2 || ec.Args.Default[0] != "-w" {
		t.Errorf("expected overridden args, got %v", ec.Args.Default)
	}
}

func TestMatrixResolverIntegration(t *testing.T) {
	// Create mock config with custom editor and set it to global callback
	cfg := defaultConfig()
	cfg.User.Editors = map[string]EditorConfig{
		"myeditor": {
			Cmd:  "myeditor-bin",
			Type: "vscode",
		},
		"cli-editor": {
			Cmd:  "cli-bin",
			Type: "cli",
		},
	}

	// Register a mock config resolver helper to force returning this cfg during test
	originalResolver := matrix.EditorTypeResolver
	defer func() {
		matrix.EditorTypeResolver = originalResolver
	}()

	matrix.EditorTypeResolver = func(ed detect.Editor) string {
		if ec, ok := cfg.ResolveEditor(string(ed)); ok {
			return ec.Type
		}
		return ""
	}

	// Test capability resolution
	cap := matrix.ResolveCapability(detect.OSWindows, detect.Editor("myeditor"))
	if !cap.CanTryDevcontainerAttach {
		t.Error("expected custom vscode editor to support devcontainer attach")
	}
	if cap.LocalOpenLevel != matrix.L1Supported {
		t.Errorf("expected L1Supported, got %v", cap.LocalOpenLevel)
	}

	capCLI := matrix.ResolveCapability(detect.OSWindows, detect.Editor("cli-editor"))
	if capCLI.CanTryDevcontainerAttach {
		t.Error("expected custom cli editor NOT to support devcontainer attach")
	}
	if !capCLI.CanRunClaudeLocally {
		t.Error("expected custom cli editor to run locally (Claude-like)")
	}

	capUnknown := matrix.ResolveCapability(detect.OSWindows, detect.Editor("non-existent"))
	if capUnknown.CanTryDevcontainerAttach {
		t.Error("expected unknown editor to fail devcontainer attach support")
	}
}
