package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const featureName = "integration-test"
const branchName = "integration-test"

// projectRoot returns the absolute path to the project root.
// Derived from this file's location: tests/integration-test/ -> 2 levels up.
func projectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller information")
	}
	// helpers_test.go is in tests/tt/
	dir := filepath.Dir(filename)
	root, err := filepath.Abs(filepath.Join(dir, "..", ".."))
	if err != nil {
		panic(fmt.Sprintf("failed to resolve project root: %v", err))
	}
	return root
}

// ttBinary returns the path to the tt binary.
// Fails the test if the binary is not found.
// On Windows, if the binary doesn't have .exe extension, creates a copy
// with .exe extension since exec.Command requires it.
func ttBinary(t *testing.T) string {
	t.Helper()

	exePath := filepath.Join(projectRoot(), "bin", "tt.exe")
	noExtPath := filepath.Join(projectRoot(), "bin", "tt")

	// Check if .exe version exists
	if _, err := os.Stat(exePath); err == nil {
		return exePath
	}

	// Check if non-.exe version exists
	if _, err := os.Stat(noExtPath); err == nil {
		if runtime.GOOS == "windows" {
			// Windows exec.Command requires .exe — copy the binary
			data, err := os.ReadFile(noExtPath)
			if err != nil {
				t.Fatalf("failed to read tt binary: %v", err)
			}
			if err := os.WriteFile(exePath, data, 0o755); err != nil {
				t.Fatalf("failed to create tt.exe: %v", err)
			}
			t.Logf("Created %s from %s for Windows compatibility", exePath, noExtPath)
			return exePath
		}
		return noExtPath
	}

	t.Fatalf("tt binary not found at %s or %s. Run ./scripts/process/build.sh first.", exePath, noExtPath)
	return ""
}

// requireDockerAvailable verifies that Docker is accessible.
// Fails the test if Docker is not available.
func requireDockerAvailable(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Docker is not available. 'docker info' failed: %v\nOutput: %s", err, string(out))
	}
}

// runTT executes the tt binary with the given arguments.
// Returns stdout, stderr, and exit code.
func runTT(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	binary := ttBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = projectRoot()

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		} else {
			// Context timeout or other error
			exitCode = -1
		}
	}

	return stdout, stderr, exitCode
}

// runTTWithInput executes the tt binary with stdin content.
// Returns stdout, stderr, and exit code.
func runTTWithInput(t *testing.T, input string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	binary := ttBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = projectRoot()
	cmd.Stdin = strings.NewReader(input)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if err != nil {
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return stdout, stderr, exitCode
}

// ensureWorktree ensures that the worktree exists for the branch.
// This is required because 'tt up' no longer creates worktrees automatically.
// Does not fail on errors (the worktree may already exist).
func ensureWorktree(t *testing.T) {
	t.Helper()
	binary := ttBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "create", branchName)
	cmd.Dir = projectRoot()
	_ = cmd.Run()
}

// cleanupTTDown runs 'tt down' to clean up containers.
// Does not fail on errors (used for cleanup, not assertions).
func cleanupTTDown(t *testing.T) {
	t.Helper()
	binary := ttBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "down", branchName, featureName)
	cmd.Dir = projectRoot()
	_ = cmd.Run()
}

// dockerRun executes a docker command and returns stdout.
func dockerRun(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// cleanupContainers removes any containers and images created during testing.
func cleanupContainers() {
	// Remove containers matching the feature name
	out, err := dockerRun("ps", "-a", "--filter", fmt.Sprintf("name=%s", featureName), "--format", "{{.Names}}")
	if err == nil {
		for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
			if name != "" {
				_, _ = dockerRun("rm", "-f", name)
			}
		}
	}

	// Remove verification images
	for _, img := range []string{"integration-test-verify", "tt-verify"} {
		_, _ = dockerRun("rmi", "-f", img)
	}
}

// TestMain runs all tests and performs cleanup afterward.
func TestMain(m *testing.M) {
	code := m.Run()
	cleanupContainers()
	os.Exit(code)
}
