package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptTagsList_ListsBaselineAndExternal(t *testing.T) {
	tmp := t.TempDir()
	writeTaggedEntities(t, tmp)

	capDir := filepath.Join(tmp, "prompts", "manifest", "code_content", "capabilities")
	ext := `---
apiVersion: agent.meta/v1
kind: capability
id: acs-external-agent-sample
title: Ext
description: sample
body: inline
tags:
  - external-agent
---

# Ext
`
	require.NoError(t, os.WriteFile(filepath.Join(capDir, "acs-external-agent-sample.md"), []byte(ext), 0o644))

	// minimal schemas dir so compile path works if needed — tags list only parses
	schemaSrc := filepath.Join(projectRoot(), "prompts", "manifest", "schemas")
	schemaDst := filepath.Join(tmp, "prompts", "manifest", "schemas")
	require.NoError(t, copyDir(schemaSrc, schemaDst))

	stdout, stderr, code := runTTInDir(t, tmp, "prompt", "tags",
		"--workspace", tmp,
		"--project", filepath.Join(tmp, "prompts", "manifest", "project.yaml"),
	)
	require.Equalf(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	require.Contains(t, stdout, "baseline")
	require.Contains(t, stdout, "external-agent")
	require.Contains(t, stdout, "acs-external-agent-sample")
	require.Contains(t, stdout, "ACS selection tags")
}

func TestPromptHelpMentionsACS(t *testing.T) {
	stdout, stderr, code := runTT(t, "prompt", "--help")
	require.Equalf(t, 0, code, "stderr=%s", stderr)
	combined := stdout + stderr
	require.True(t, strings.Contains(combined, "ACS") || strings.Contains(combined, "tags"),
		"help should mention ACS or tags: %s", combined)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
