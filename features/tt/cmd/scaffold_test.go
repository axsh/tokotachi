package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRepoRoot(t *testing.T) {
	// 1. When rootPath is specified, it should return the specified path as is.
	assert.Equal(t, "/some/root/path", resolveRepoRoot("/some/root/path"))

	// 2. When rootPath is empty, it should return the current working directory.
	wd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, wd, resolveRepoRoot(""))
}
