package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveGitLab_Doctor_ChecksGlab(t *testing.T) {
	baseURL, token, _ := requireLiveGitLab(t)
	dir := t.TempDir()

	runGitIn(t, dir, "init")
	origin := baseURL + "/" + defaultProject + ".git"
	runGitIn(t, dir, "remote", "add", "origin", origin)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "features", featureDemoName), 0o755))

	stdout, stderr, code := runTTInDirEnv(t, dir, liveEnv(t, token, glabNIPath(t)), "doctor", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "glab")
	assert.NotContains(t, names, "gh")
}
