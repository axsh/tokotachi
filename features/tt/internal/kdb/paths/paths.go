// Package paths resolves workspace, registry, and per-KDB filesystem locations.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/registry"
)

// DefaultKdbRoot is the workspace-relative directory that holds registry.yaml
// and per-KDB modules when no --kdb-root / KDB_ROOT override is set.
const DefaultKdbRoot = "shared/kdb"

// ResolveInput carries explicit overrides (flags / BL_* environment values).
type ResolveInput struct {
	Workspace string // flag / TT_WORKSPACE / BL_ROOT / cwd
	Registry  string // flag / KDB_REGISTRY / {workspace}/{kdb-root}/registry.yaml
	KdbRoot   string // flag / KDB_ROOT / DefaultKdbRoot
	// Backlog overrides (BL_*):
	DataDir, DBPath, CorpusDir, TmpDir string
	Today                              string // BL_TODAY YYYY-MM-DD; empty -> UTC today
}

// Paths holds resolved absolute locations for one KDB.
type Paths struct {
	Workspace    string
	RegistryFile string
	Entry        registry.Entry
	ModuleDir    string
	CorpusDir    string
	DataDir      string
	DBPath       string
	TmpDir       string
	DataJSON     string
	TmpJSON      string
	Today        string
}

// ResolveWorkspace returns an absolute workspace root.
// Priority: flag > TT_WORKSPACE > BL_ROOT > cwd.
func ResolveWorkspace(flagWorkspace string) (string, error) {
	candidates := []string{flagWorkspace, os.Getenv("TT_WORKSPACE"), os.Getenv("BL_ROOT")}
	for _, c := range candidates {
		if c != "" {
			return filepath.Abs(c)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	return wd, nil
}

// ResolveKdbRoot returns the workspace-relative KDB root directory.
// Priority: flag > KDB_ROOT > DefaultKdbRoot.
func ResolveKdbRoot(flagKdbRoot string) string {
	root := flagKdbRoot
	if root == "" {
		root = os.Getenv("KDB_ROOT")
	}
	if root == "" {
		root = DefaultKdbRoot
	}
	root = filepath.ToSlash(filepath.Clean(root))
	root = strings.TrimPrefix(root, "./")
	return strings.TrimSuffix(root, "/")
}

// RegistryPath returns the registry file path.
// Priority: flag > KDB_REGISTRY > {workspace}/{kdbRoot}/registry.yaml.
func RegistryPath(workspace, flagRegistry, kdbRoot string) string {
	if flagRegistry != "" {
		return flagRegistry
	}
	if env := os.Getenv("KDB_REGISTRY"); env != "" {
		return env
	}
	return filepath.Join(workspace, filepath.FromSlash(ResolveKdbRoot(kdbRoot)), "registry.yaml")
}

// BuiltinBacklogEntry returns the built-in backlog registry entry for kdbRoot.
func BuiltinBacklogEntry(kdbRoot string) registry.Entry {
	root := ResolveKdbRoot(kdbRoot)
	module := root + "/backlog"
	return registry.Entry{
		ID:         "backlog",
		ModulePath: module,
		CorpusPath: module + "/corpus",
		StatusCmd:  "tt kdb backlog status",
		DistillCmd: "tt kdb backlog distill",
		DumpCmd:    "tt kdb backlog dump",
		MergeCmd:   "tt kdb backlog merge",
		PackCmd:    "tt kdb backlog pack",
	}
}

// ForEntry resolves all paths for a registry entry.
func ForEntry(in ResolveInput, e registry.Entry) (Paths, error) {
	ws, err := ResolveWorkspace(in.Workspace)
	if err != nil {
		return Paths{}, err
	}
	p := Paths{
		Workspace:    ws,
		RegistryFile: RegistryPath(ws, in.Registry, in.KdbRoot),
		Entry:        e,
	}
	if e.ModulePath != "" {
		p.ModuleDir = filepath.Join(ws, filepath.FromSlash(e.ModulePath))
	}

	if in.CorpusDir != "" {
		p.CorpusDir = absFrom(ws, in.CorpusDir)
	} else if e.CorpusPath != "" {
		p.CorpusDir = filepath.Join(ws, filepath.FromSlash(e.CorpusPath))
	}

	if in.DataDir != "" {
		p.DataDir = absFrom(ws, in.DataDir)
	} else if p.CorpusDir != "" {
		p.DataDir = filepath.Join(filepath.Dir(p.CorpusDir), "data")
	} else if p.ModuleDir != "" {
		p.DataDir = filepath.Join(p.ModuleDir, "data")
	}

	if in.TmpDir != "" {
		p.TmpDir = absFrom(ws, in.TmpDir)
	} else {
		p.TmpDir = filepath.Join(ws, "tmp", "kdb", e.ID)
	}

	if in.DBPath != "" {
		p.DBPath = absFrom(ws, in.DBPath)
	} else if e.ID == "backlog" && p.DataDir != "" {
		p.DBPath = filepath.Join(p.DataDir, "backlog.sqlite")
	}

	if p.DataDir != "" {
		p.DataJSON = filepath.Join(p.DataDir, "items.json")
	}
	p.TmpJSON = filepath.Join(p.TmpDir, "items.json")

	p.Today = in.Today
	if p.Today == "" {
		p.Today = time.Now().UTC().Format("2006-01-02")
	}
	return p, nil
}

// InputFromEnv builds a ResolveInput from the BL_* / KDB_* environment variables,
// layered under explicit flag values.
func InputFromEnv(workspace, registryFile, kdbRoot string) ResolveInput {
	return ResolveInput{
		Workspace: workspace,
		Registry:  registryFile,
		KdbRoot:   kdbRoot,
		DataDir:   os.Getenv("BL_DATA_DIR"),
		DBPath:    os.Getenv("BL_DB_PATH"),
		CorpusDir: os.Getenv("BL_CORPUS_DIR"),
		TmpDir:    os.Getenv("BL_TMP_DIR"),
		Today:     os.Getenv("BL_TODAY"),
	}
}

func absFrom(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}
