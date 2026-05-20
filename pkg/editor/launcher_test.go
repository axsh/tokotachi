package editor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCustomLauncher_BindPlaceholders(t *testing.T) {
	l := &CustomLauncher{
		name: "test",
		cfg:  EditorConfig{},
	}

	// Mock HOME directory
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	mockHome := "/mock/home"
	if runtime.GOOS == "windows" {
		mockHome = `C:\mock\home`
	}
	os.Setenv("HOME", mockHome)
	os.Setenv("USERPROFILE", mockHome)

	os.Setenv("LOCALAPPDATA", "/mock/localappdata")
	defer os.Unsetenv("LOCALAPPDATA")

	resolvedHome := l.bindPlaceholders("{home}/bin/app")
	expectedHome := filepath.Join(mockHome, "bin/app")
	if resolvedHome != expectedHome {
		t.Errorf("expected %q, got %q", expectedHome, resolvedHome)
	}

	resolvedLocal := l.bindPlaceholders("{localappdata}/bin/app")
	expectedLocal := filepath.Clean("/mock/localappdata/bin/app")
	if resolvedLocal != expectedLocal {
		t.Errorf("expected %q, got %q", expectedLocal, resolvedLocal)
	}
}

func TestCustomLauncher_ResolveCommand(t *testing.T) {
	// Create a temp directory and write a dummy executable file to mock path detection
	tmpDir, err := os.MkdirTemp("", "tokotachi-launcher-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dummyFile := filepath.Join(tmpDir, "dummy-editor.exe")
	if err := os.WriteFile(dummyFile, []byte(""), 0755); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	l := &CustomLauncher{
		name: "test",
		cfg:  EditorConfig{},
	}

	// 1. If only cmd is provided, it should resolve it
	resolved := l.resolveCommand("cmd-only", nil)
	if resolved != "cmd-only" {
		t.Errorf("expected 'cmd-only', got %q", resolved)
	}

	// 2. If cmds is provided, it should check them in priority order
	// Since "nonexistent" doesn't exist, it should fallback to dummyFile which exists directly
	cmds := []string{"nonexistent-cmd", dummyFile}
	resolved = l.resolveCommand("primary", cmds)
	if resolved != dummyFile {
		t.Errorf("expected %q, got %q", dummyFile, resolved)
	}

	// 3. If none of the candidates exist, it should fallback to the first one in the list
	cmds = []string{"nonexistent-1", "nonexistent-2"}
	resolved = l.resolveCommand("primary", cmds)
	if resolved != "nonexistent-1" {
		t.Errorf("expected 'nonexistent-1', got %q", resolved)
	}
}
