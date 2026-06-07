package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
)

func TestConvert(t *testing.T) {
	t.Run("full pipeline execution", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set up a realistic project structure.
		// go.mod
		mustWriteFile(t, filepath.Join(tmpDir, "go.mod"),
			"module old-org/old-app\n\ngo 1.24.0\n")

		// .git directory (should be cleaned up)
		mustMkdir(t, filepath.Join(tmpDir, ".git"))
		mustWriteFile(t, filepath.Join(tmpDir, ".git", "config"), "gitconfig")

		// go.sum (should be cleaned up)
		mustWriteFile(t, filepath.Join(tmpDir, "go.sum"), "checksum data")

		// cmd/old-app/main.go (has import to transform)
		mustMkdir(t, filepath.Join(tmpDir, "cmd", "old-app"))
		mustWriteFile(t, filepath.Join(tmpDir, "cmd", "old-app", "main.go"),
			"package main\n\nimport \"old-org/old-app/internal/pkg\"\n\nfunc main() {}\n")

		// internal/pkg/pkg.go (no import to transform)
		mustMkdir(t, filepath.Join(tmpDir, "internal", "pkg"))
		mustWriteFile(t, filepath.Join(tmpDir, "internal", "pkg", "pkg.go"),
			"package pkg\n\nfunc Hello() string { return \"hello\" }\n")

		// Makefile + Makefile.hints
		mustWriteFile(t, filepath.Join(tmpDir, "Makefile"),
			"APP=old-app\nMODULE=old-org/old-app\n")
		mustWriteFile(t, filepath.Join(tmpDir, "Makefile.hints"),
			"replacements:\n  - match: \"old-app\"\n    replace_with: \"{{feature_name}}\"\n  - match: \"old-org/old-app\"\n    replace_with: \"{{module_path}}\"\n")

		params := ConvertParams{
			OldModule:  "old-org/old-app",
			NewModule:  "{{module_path}}/{{feature_name}}",
			OldProgram: "old-app",
			NewProgram: "new-app",
			HintParams: map[string]string{
				"feature_name": "new-app",
				"module_path":  "github.com/new-org/new-app",
			},
		}

		pc, err := Convert(tmpDir, params)
		if err != nil {
			t.Fatalf("Convert() error: %v", err)
		}

		// Step1: Cleanup — .git and go.sum should be removed
		assertNotExists(t, tmpDir, ".git")
		assertNotExists(t, tmpDir, "go.sum")

		// Step2: AST transform — go.mod.tmpl should exist
		assertExists(t, tmpDir, "go.mod.tmpl")
		assertNotExists(t, tmpDir, "go.mod")
		goModContent := mustReadFile(t, filepath.Join(tmpDir, "go.mod.tmpl"))
		assertContains(t, goModContent, "module {{module_path}}/{{feature_name}}")

		// Step2: AST transform — main.go.tmpl should exist with transformed import
		assertExists(t, tmpDir, filepath.Join("cmd", "new-app", "main.go.tmpl"))
		mainContent := mustReadFile(t, filepath.Join(tmpDir, "cmd", "new-app", "main.go.tmpl"))
		assertContains(t, mainContent, "{{module_path}}/{{feature_name}}/internal/pkg")

		// Step2: non-transformed file should remain without .tmpl
		assertExists(t, tmpDir, filepath.Join("internal", "pkg", "pkg.go"))
		assertNotExists(t, tmpDir, filepath.Join("internal", "pkg", "pkg.go.tmpl"))

		// Step3: Directory rename — cmd/old-app should be gone
		assertNotExists(t, tmpDir, filepath.Join("cmd", "old-app"))
		assertExists(t, tmpDir, filepath.Join("cmd", "new-app"))

		// Step4: Hints — Makefile.tmpl should exist with replacements applied
		assertExists(t, tmpDir, "Makefile.tmpl")
		assertNotExists(t, tmpDir, "Makefile")
		assertNotExists(t, tmpDir, "Makefile.hints")
		makefileContent := mustReadFile(t, filepath.Join(tmpDir, "Makefile.tmpl"))
		assertContains(t, makefileContent, "APP=new-app")
		assertContains(t, makefileContent, "MODULE=github.com/new-org/new-app")

		// Verify ParamCollector collected template vars
		names := pc.Names()
		if len(names) < 2 {
			t.Fatalf("ParamCollector should have at least 2 params, got %v", names)
		}
		assertSliceContains(t, names, "feature_name")
		assertSliceContains(t, names, "module_path")
	})

	t.Run("real-world scaffold: go.mod module mismatch with oldModule", func(t *testing.T) {
		tmpDir := t.TempDir()

		// go.mod with a module path that differs from OldModule (scaffold.yaml default).
		mustWriteFile(t, filepath.Join(tmpDir, "go.mod"),
			"module github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature\n\ngo 1.24.0\n")

		// main.go with import using the actual go.mod module path.
		mustWriteFile(t, filepath.Join(tmpDir, "main.go"),
			"package main\n\nimport \"github.com/axsh/tokotachi-scaffolds/axsh/go-standard-feature/internal/pkg\"\n\nfunc main() {}\n")

		// internal/pkg/pkg.go (no import to transform)
		mustMkdir(t, filepath.Join(tmpDir, "internal", "pkg"))
		mustWriteFile(t, filepath.Join(tmpDir, "internal", "pkg", "pkg.go"),
			"package pkg\n\nfunc Hello() string { return \"hello\" }\n")

		params := ConvertParams{
			OldModule:  "github.com/axsh/tokotachi/features/myprog", // scaffold.yaml default (does NOT match go.mod)
			NewModule:  "{{module_path}}/{{feature_name}}",
			OldProgram: "myprog",
			NewProgram: "myprog",
			HintParams: map[string]string{
				"feature_name": "myprog",
				"module_path":  "github.com/axsh/tokotachi/features/myprog",
			},
		}

		pc, err := Convert(tmpDir, params)
		if err != nil {
			t.Fatalf("Convert() error: %v", err)
		}

		// go.mod.tmpl should exist with template variable.
		assertExists(t, tmpDir, "go.mod.tmpl")
		assertNotExists(t, tmpDir, "go.mod")
		goModContent := mustReadFile(t, filepath.Join(tmpDir, "go.mod.tmpl"))
		assertContains(t, goModContent, "module {{module_path}}/{{feature_name}}")

		// main.go.tmpl should exist with import transformed using discovered module path.
		assertExists(t, tmpDir, "main.go.tmpl")
		assertNotExists(t, tmpDir, "main.go")
		mainContent := mustReadFile(t, filepath.Join(tmpDir, "main.go.tmpl"))
		assertContains(t, mainContent, "{{module_path}}/{{feature_name}}/internal/pkg")

		// non-transformed file should remain without .tmpl
		assertExists(t, tmpDir, filepath.Join("internal", "pkg", "pkg.go"))
		assertNotExists(t, tmpDir, filepath.Join("internal", "pkg", "pkg.go.tmpl"))

		// Verify ParamCollector
		names := pc.Names()
		assertSliceContains(t, names, "feature_name")
		assertSliceContains(t, names, "module_path")
	})

	t.Run("no template_params skips conversion", func(t *testing.T) {
		tmpDir := t.TempDir()
		mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "package main\nfunc main() {}\n")

		params := ConvertParams{}

		_, err := Convert(tmpDir, params)
		if err != nil {
			t.Fatalf("Convert() error: %v", err)
		}

		// File should still exist unchanged (no .tmpl)
		assertExists(t, tmpDir, "main.go")
	})
}

func TestBuildConvertParamsOldValueFallback(t *testing.T) {
	t.Run("falls back to default when old_value is empty", func(t *testing.T) {
		params := BuildConvertParams([]catalog.TemplateParam{
			{
				Name:     "module_path",
				Default:  "github.com/axsh/tokotachi/features/myprog",
				OldValue: "", // empty → should fallback to Default
			},
			{
				Name:     "feature_name",
				Default:  "myprog",
				OldValue: "", // empty → should fallback to Default
			},
		})

		if params.OldModule != "github.com/axsh/tokotachi/features/myprog" {
			t.Errorf("OldModule = %q, want %q", params.OldModule, "github.com/axsh/tokotachi/features/myprog")
		}
		if params.OldProgram != "myprog" {
			t.Errorf("OldProgram = %q, want %q", params.OldProgram, "myprog")
		}
		if params.HintParams["module_path"] != "github.com/axsh/tokotachi/features" {
			t.Errorf("HintParams[module_path] = %q, want %q", params.HintParams["module_path"], "github.com/axsh/tokotachi/features")
		}
		if params.NewModule != "{{module_path}}/{{feature_name}}" {
			t.Errorf("NewModule = %q, want %q", params.NewModule, "{{module_path}}/{{feature_name}}")
		}
	})

	t.Run("explicit old_value takes priority over default", func(t *testing.T) {
		params := BuildConvertParams([]catalog.TemplateParam{
			{
				Name:     "module_path",
				Default:  "something-else",
				OldValue: "function", // explicit → should be used
			},
			{
				Name:     "feature_name",
				Default:  "other-name",
				OldValue: "function", // explicit → should be used
			},
		})

		if params.OldModule != "function" {
			t.Errorf("OldModule = %q, want %q", params.OldModule, "function")
		}
		if params.OldProgram != "function" {
			t.Errorf("OldProgram = %q, want %q", params.OldProgram, "function")
		}
		if params.NewModule != "{{module_path}}/{{feature_name}}" {
			t.Errorf("NewModule = %q, want %q", params.NewModule, "{{module_path}}/{{feature_name}}")
		}
	})

	t.Run("module_path only without feature_name", func(t *testing.T) {
		params := BuildConvertParams([]catalog.TemplateParam{
			{
				Name:    "module_path",
				Default: "github.com/org/app",
			},
		})

		if params.NewModule != "{{module_path}}" {
			t.Errorf("NewModule = %q, want %q", params.NewModule, "{{module_path}}")
		}
	})

	t.Run("module_path suffix matches feature_name — suffix stripped for HintParams", func(t *testing.T) {
		params := BuildConvertParams([]catalog.TemplateParam{
			{
				Name:    "module_path",
				Default: "github.com/org/features/myapp",
			},
			{
				Name:    "feature_name",
				Default: "myapp",
			},
		})

		if params.OldModule != "github.com/org/features/myapp" {
			t.Errorf("OldModule = %q, want %q", params.OldModule, "github.com/org/features/myapp")
		}
		if params.HintParams["module_path"] != "github.com/org/features" {
			t.Errorf("HintParams[module_path] = %q, want %q", params.HintParams["module_path"], "github.com/org/features")
		}
	})

	t.Run("module_path suffix does not match feature_name — no stripping", func(t *testing.T) {
		params := BuildConvertParams([]catalog.TemplateParam{
			{
				Name:    "module_path",
				Default: "github.com/org/app",
			},
			{
				Name:    "feature_name",
				Default: "other",
			},
		})

		if params.OldModule != "github.com/org/app" {
			t.Errorf("OldModule = %q, want %q", params.OldModule, "github.com/org/app")
		}
		if params.HintParams["module_path"] != "github.com/org/app" {
			t.Errorf("HintParams[module_path] = %q, want %q", params.HintParams["module_path"], "github.com/org/app")
		}
	})
}

// --- Test helpers ---

func assertExists(t *testing.T, base string, relPath string) {
	t.Helper()
	p := filepath.Join(base, relPath)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		t.Errorf("expected %q to exist, but it does not", relPath)
	}
}

func assertNotExists(t *testing.T, base string, relPath string) {
	t.Helper()
	p := filepath.Join(base, relPath)
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("expected %q to NOT exist, but it does", relPath)
	}
}

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !containsStr(content, substr) {
		t.Errorf("expected content to contain %q, got:\n%s", substr, content)
	}
}

func assertSliceContains(t *testing.T, slice []string, want string) {
	t.Helper()
	for _, s := range slice {
		if s == want {
			return
		}
	}
	t.Errorf("expected slice %v to contain %q", slice, want)
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	return string(data)
}
