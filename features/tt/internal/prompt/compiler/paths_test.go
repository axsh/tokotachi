package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataValidProject(t *testing.T) (cwd, projectRel string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller")
	}
	dir := filepath.Dir(filename)
	cwd = filepath.Join(dir, "testdata", "valid")
	projectRel = filepath.Join("prompts", "manifest", "project.yaml")
	return cwd, projectRel
}

func TestResolvePaths_DefaultBackwardCompatible(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    projectRel,
		ProjectFlagSet: true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	wantWorkspace := cwd
	if paths.Workspace != wantWorkspace {
		t.Errorf("Workspace = %q, want %q", paths.Workspace, wantWorkspace)
	}

	wantProject := filepath.Join(cwd, "prompts", "manifest", "project.yaml")
	if paths.ProjectYAML != wantProject {
		t.Errorf("ProjectYAML = %q, want %q", paths.ProjectYAML, wantProject)
	}

	if paths.PromptsDir != "prompts" {
		t.Errorf("PromptsDir = %q, want prompts", paths.PromptsDir)
	}

	wantBuild := filepath.Join(cwd, "tmp", "dist")
	if paths.BuildDirAbs() != wantBuild {
		t.Errorf("BuildDirAbs() = %q, want %q", paths.BuildDirAbs(), wantBuild)
	}

	wantSchemas := filepath.Join(cwd, "prompts", "manifest", "schemas")
	if paths.SchemasDir != wantSchemas {
		t.Errorf("SchemasDir = %q, want %q", paths.SchemasDir, wantSchemas)
	}

	if paths.DeployRoot != wantWorkspace {
		t.Errorf("DeployRoot = %q, want %q", paths.DeployRoot, wantWorkspace)
	}
}

func TestResolvePaths_WorkspaceFlag(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:              cwd,
		WorkspaceFlag:    ".",
		WorkspaceFlagSet: true,
		ProjectFlag:      projectRel,
		ProjectFlagSet:   true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	wantProject := filepath.Join(cwd, "prompts", "manifest", "project.yaml")
	if paths.ProjectYAML != wantProject {
		t.Errorf("ProjectYAML = %q, want %q", paths.ProjectYAML, wantProject)
	}
}

func TestResolvePaths_PromptsDirFlag(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:               cwd,
		PromptsDirFlag:    "catalog/prompts",
		PromptsDirFlagSet: true,
		ProjectFlag:       projectRel,
		ProjectFlagSet:    true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	wantSchemas := filepath.Join(cwd, "catalog", "prompts", "manifest", "schemas")
	if paths.SchemasDir != wantSchemas {
		t.Errorf("SchemasDir = %q, want %q", paths.SchemasDir, wantSchemas)
	}

	watch := paths.GitWatchPaths()
	if len(watch) != 2 {
		t.Fatalf("GitWatchPaths() len = %d, want 2", len(watch))
	}
	if watch[0] != "catalog/prompts/manifest" {
		t.Errorf("GitWatchPaths()[0] = %q, want catalog/prompts/manifest", watch[0])
	}
	if watch[1] != "catalog/prompts/memory" {
		t.Errorf("GitWatchPaths()[1] = %q, want catalog/prompts/memory", watch[1])
	}
}

func TestResolvePaths_BuildDirFlag(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:             cwd,
		BuildDirFlag:    ".cache/tt-dist",
		BuildDirFlagSet: true,
		ProjectFlag:     projectRel,
		ProjectFlagSet:  true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	want := filepath.Join(cwd, ".cache", "tt-dist")
	if paths.BuildDirAbs() != want {
		t.Errorf("BuildDirAbs() = %q, want %q", paths.BuildDirAbs(), want)
	}
}

func TestResolvePaths_ResolvedManifestFlag(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:                     cwd,
		ResolvedManifestFlag:    "out/custom.yaml",
		ResolvedManifestFlagSet: true,
		ProjectFlag:             projectRel,
		ProjectFlagSet:          true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	want := filepath.Join(cwd, "out", "custom.yaml")
	if paths.ResolvedManifestAbs() != want {
		t.Errorf("ResolvedManifestAbs() = %q, want %q", paths.ResolvedManifestAbs(), want)
	}
}

func TestResolvePaths_BuildDirWithoutResolvedManifest(t *testing.T) {
	cwd, _ := testdataValidProject(t)
	tmp := t.TempDir()
	copyTestdata(t, cwd, tmp)

	projectRel := filepath.Join("prompts", "manifest", "project.yaml")
	// Remove resolved_manifest from effective yaml by using env override empty - actually
	// use custom project in temp with empty outputs - simpler: set build dir only and
	// clear yaml resolved via env empty won't work. Use ResolvedManifestFlagSet false
	// and modify - the valid testdata has resolved_manifest set. For fallback test,
	// set TT_RESOLVED_MANIFEST to empty and BuildDirFlag - merge won't clear yaml.

	// Create minimal project without resolved_manifest output
	projDir := filepath.Join(tmp, "prompts", "manifest")
	projectPath := filepath.Join(projDir, "project.yaml")
	if err := os.WriteFile(projectPath, []byte(`version: 1
project_id: x
sources:
  policies: "prompts/manifest/code_content/policies/**/*.yaml"
outputs:
  resolved_manifest: ""
defaults:
  build_dir: tmp/dist/
`), 0644); err != nil {
		t.Fatal(err)
	}

	paths, err := ResolvePaths(PathOptions{
		CWD:             tmp,
		BuildDirFlag:    ".cache/tt-dist",
		BuildDirFlagSet: true,
		ProjectFlag:     projectRel,
		ProjectFlagSet:  true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	want := filepath.Join(tmp, ".cache", "tt-dist", "manifest.resolved.yaml")
	if paths.ResolvedManifestAbs() != want {
		t.Errorf("ResolvedManifestAbs() = %q, want %q", paths.ResolvedManifestAbs(), want)
	}
}

func TestResolvePaths_DeployRootFlag(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)
	deployRoot := t.TempDir()

	paths, err := ResolvePaths(PathOptions{
		CWD:               cwd,
		DeployRootFlag:    deployRoot,
		DeployRootFlagSet: true,
		ProjectFlag:       projectRel,
		ProjectFlagSet:    true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.DeployRoot != deployRoot {
		t.Errorf("DeployRoot = %q, want %q", paths.DeployRoot, deployRoot)
	}
	if paths.Workspace == deployRoot && cwd != deployRoot {
		// workspace and deploy root should differ when deploy root is external temp
		if paths.Workspace != cwd {
			t.Errorf("Workspace = %q, want %q", paths.Workspace, cwd)
		}
	}
}

func TestResolvePaths_PriorityFlagOverEnv(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)
	t.Setenv("TT_BUILD_DIR", "env-dist")

	paths, err := ResolvePaths(PathOptions{
		CWD:             cwd,
		BuildDirFlag:    "cli-dist",
		BuildDirFlagSet: true,
		ProjectFlag:     projectRel,
		ProjectFlagSet:  true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.BuildDir != "cli-dist" {
		t.Errorf("BuildDir = %q, want cli-dist", paths.BuildDir)
	}
}

func TestResolvePaths_PriorityEnvOverYAML(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)
	t.Setenv("TT_BUILD_DIR", "env-dist")

	paths, err := ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    projectRel,
		ProjectFlagSet: true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.BuildDir != "env-dist" {
		t.Errorf("BuildDir = %q, want env-dist", paths.BuildDir)
	}
}

func TestResolvePaths_AbsolutePathIgnoresAnchor(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)
	absDeploy := t.TempDir()

	paths, err := ResolvePaths(PathOptions{
		CWD:               cwd,
		DeployRootFlag:    absDeploy,
		DeployRootFlagSet: true,
		ProjectFlag:       projectRel,
		ProjectFlagSet:    true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.DeployRoot != absDeploy {
		t.Errorf("DeployRoot = %q, want %q", paths.DeployRoot, absDeploy)
	}
}

func TestResolvePaths_ProjectRelativeToWorkspace(t *testing.T) {
	cwd, _ := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:              cwd,
		WorkspaceFlag:    cwd,
		WorkspaceFlagSet: true,
		ProjectFlag:      "prompts/manifest/project.yaml",
		ProjectFlagSet:   true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	want := filepath.Join(cwd, "prompts", "manifest", "project.yaml")
	if paths.ProjectYAML != want {
		t.Errorf("ProjectYAML = %q, want %q", paths.ProjectYAML, want)
	}
}

func TestResolvePaths_SchemasDir(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    projectRel,
		ProjectFlagSet: true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	want := filepath.Join(cwd, "prompts", "manifest", "schemas")
	if paths.SchemasDir != want {
		t.Errorf("SchemasDir = %q, want %q", paths.SchemasDir, want)
	}
}

func TestResolvePaths_GitWatchPaths(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    projectRel,
		ProjectFlagSet: true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	watch := paths.GitWatchPaths()
	if watch[0] != "prompts/manifest" || watch[1] != "prompts/memory" {
		t.Errorf("GitWatchPaths() = %v, want [prompts/manifest prompts/memory]", watch)
	}
}

func TestResolvePaths_InferredWorkspaceFromProject(t *testing.T) {
	cwd, projectRel := testdataValidProject(t)

	paths, err := ResolvePaths(PathOptions{
		CWD:            cwd,
		ProjectFlag:    projectRel,
		ProjectFlagSet: true,
	})
	if err != nil {
		t.Fatalf("ResolvePaths() error = %v", err)
	}

	if paths.Workspace != cwd {
		t.Errorf("Workspace = %q, want inferred %q", paths.Workspace, cwd)
	}
}
