package integration_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIntegrationTestDockerfileBuild verifies that the integration-test
// feature Dockerfile builds successfully.
func TestIntegrationTestDockerfileBuild(t *testing.T) {
	requireDockerAvailable(t)

	buildContext := filepath.Join(projectRoot(), "features", "integration-test")
	dockerfile := filepath.Join(buildContext, ".devcontainer", "Dockerfile")
	imageName := "integration-test-verify"

	cmd := exec.Command("docker", "build", "-f", dockerfile, "-t", imageName, buildContext)
	out, err := cmd.CombinedOutput()

	assert.NoError(t, err, "Docker build failed for integration-test.\nOutput:\n%s", string(out))

	// Cleanup
	_, _ = dockerRun("rmi", "-f", imageName)
}

// TestTtDockerfileBuild verifies that the tt feature Dockerfile
// builds successfully. Requires golang:1.23+ for golangci-lint compatibility.
func TestTtDockerfileBuild(t *testing.T) {
	requireDockerAvailable(t)

	buildContext := filepath.Join(projectRoot(), "features", "tt", ".devcontainer")
	imageName := "tt-verify"

	cmd := exec.Command("docker", "build", "-t", imageName, buildContext)
	out, err := cmd.CombinedOutput()

	assert.NoError(t, err, "Docker build failed for tt.\nOutput:\n%s", string(out))

	// Cleanup
	_, _ = dockerRun("rmi", "-f", imageName)
}
