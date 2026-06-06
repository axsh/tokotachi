package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/archiver"
	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/catalog"
	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/converter"
	"github.com/axsh/tokotachi-scaffolds/features/templatizer/internal/copier"
)

// parseArgs parses CLI arguments and returns the search root directory.
// Prints usage and exits when --help/-h is given or no arguments are provided.
func parseArgs(args []string) string {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printUsage()
			os.Exit(0)
		}
	}
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}
	return args[0]
}

// printUsage prints the help message to stderr.
func printUsage() {
	fmt.Fprintf(os.Stderr, `templatizer - Scan originals and generate scaffold templates

Usage:
  templatizer <search-root-dir>
  templatizer --help

Arguments:
  <search-root-dir>  Root directory to search for 'originals' directories

Examples:
  templatizer .
  templatizer ./catalog
`)
}

func main() {
	// Parse CLI arguments.
	searchRoot := parseArgs(os.Args[1:])

	// Discover originals directory by recursive search.
	result, err := catalog.DiscoverOriginals(searchRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	baseDir := result.BaseDir
	// repoRoot is the parent of baseDir (e.g. parent of catalog/).
	// Used when joining with repo-root-relative paths like OriginalRef or ScaffoldShardPath.
	repoRoot := filepath.Dir(baseDir)
	defs := result.Definitions
	fmt.Printf("Discovered originals: %s\n", result.OriginalsDir)

	// Convert ScaffoldDefinition to Scaffold and build placement map.
	scaffolds := convertDefinitionsToScaffolds(defs)
	placements := buildPlacementMap(defs)

	// Validate dependency references and detect circular dependencies.
	tempCat := &catalog.Catalog{Scaffolds: scaffolds}
	if err := tempCat.ValidateDependencies(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: dependency validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processing %d scaffold(s)...\n", len(scaffolds))

	// Process each scaffold (template conversion + ZIP to temp).
	zipPaths := make(map[string]string) // scaffoldKey -> tempZipPath
	for i, s := range scaffolds {
		key := s.Category + "/" + s.Name
		placement := placements[key]
		tempZip, pc, procErr := processScaffold(repoRoot, s, placement)
		if procErr != nil {
			fmt.Fprintf(os.Stderr, "Error processing scaffold %q: %v\n", s.Name, procErr)
			os.Exit(1)
		}
		zipPaths[key] = tempZip

		// Merge discovered template params with scaffold.yaml definitions.
		if pc != nil && len(pc.Names()) > 0 {
			scaffolds[i].TemplateParams = converter.MergeParams(s.TemplateParams, pc.Names())
		}
	}

	// Generate shard files + move ZIPs to shard directories.
	fmt.Println("Generating shard files...")
	if err := cleanScaffoldsDir(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := generateShardFiles(repoRoot, scaffolds, zipPaths); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Write meta.yaml to repo root.
	if err := writeMetaCatalog(repoRoot, "1.0.0", "default"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Write catalog.yaml (index) to repo root.
	index := catalog.BuildCatalogIndex(scaffolds)
	if err := writeCatalogIndex(repoRoot, index); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done. All scaffolds archived and sharded successfully.")
}

// convertDefinitionsToScaffolds converts ScaffoldDefinition slice to Scaffold slice.
func convertDefinitionsToScaffolds(defs []catalog.ScaffoldDefinition) []catalog.Scaffold {
	scaffolds := make([]catalog.Scaffold, len(defs))
	for i, d := range defs {
		scaffolds[i] = catalog.Scaffold{
			Name:           d.Name,
			Category:       d.Category,
			Description:    d.Description,
			DependsOn:      d.DependsOn,
			OriginalRef:    d.OriginalRef,
			TemplateParams: d.TemplateParams,
		}
	}
	return scaffolds
}

// buildPlacementMap builds a map from scaffold key to Placement from ScaffoldDefinitions.
func buildPlacementMap(defs []catalog.ScaffoldDefinition) map[string]*catalog.Placement {
	placements := make(map[string]*catalog.Placement, len(defs))
	for i := range defs {
		key := defs[i].Category + "/" + defs[i].Name
		placements[key] = defs[i].Placement
	}
	return placements
}

// processScaffold copies originals to a temp directory, runs template conversion,
// creates a ZIP archive, and returns the path to the temp ZIP file and a ParamCollector.
// The temp ZIP file will be moved to the shard directory by generateShardFiles.
func processScaffold(repoRoot string, s catalog.Scaffold, placement *catalog.Placement) (string, *converter.ParamCollector, error) {
	originalDir := filepath.Join(repoRoot, s.OriginalRef)

	// Create temporary directory for working copy.
	tempDir, err := os.MkdirTemp("", "templatizer-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("  [%s] Copying %s -> %s\n", s.Name, originalDir, tempDir)

	// Copy originals to temp directory.
	if err := copier.CopyDir(originalDir, tempDir); err != nil {
		return "", nil, fmt.Errorf("failed to copy originals to temp: %w", err)
	}

	// Run template conversion pipeline if template_params are defined.
	var pc *converter.ParamCollector
	if len(s.TemplateParams) > 0 {
		params := converter.BuildConvertParams(s.TemplateParams)
		fmt.Printf("  [%s] Converting (module: %s -> %s)\n", s.Name, params.OldModule, params.NewModule)
		pc, err = converter.Convert(tempDir, params)
		if err != nil {
			return "", nil, fmt.Errorf("template conversion failed: %w", err)
		}
	} else {
		pc = converter.NewParamCollector()
	}

	// Collect template variables from placement.base_dir.
	if placement != nil && placement.BaseDir != "" {
		pc.AddFromString(placement.BaseDir)
	}

	// Create ZIP in a temp location.
	basename := filepath.Base(s.OriginalRef)
	tempZip := filepath.Join(os.TempDir(), fmt.Sprintf("templatizer-%s-%s.zip", s.Category, basename))
	fmt.Printf("  [%s] Archiving %s -> %s\n", s.Name, tempDir, tempZip)

	if err := archiver.ZipDirectory(tempDir, tempZip); err != nil {
		return "", nil, fmt.Errorf("failed to create ZIP archive: %w", err)
	}

	fmt.Printf("  [%s] Done (temp cleaned up)\n", s.Name)
	return tempZip, pc, nil
}

// cleanScaffoldsDir removes the existing catalog/scaffolds/ directory
// to ensure a clean state before regenerating shard files.
func cleanScaffoldsDir(repoRoot string) error {
	scaffoldsDir := filepath.Join(repoRoot, "catalog", "scaffolds")
	if err := os.RemoveAll(scaffoldsDir); err != nil {
		return fmt.Errorf("failed to clean scaffolds directory: %w", err)
	}
	return nil
}

// generateShardFiles groups scaffolds by hash, writes shard YAML files,
// moves ZIP files to shard directories, and sets template_ref.
func generateShardFiles(repoRoot string, scaffolds []catalog.Scaffold, zipPaths map[string]string) error {
	// Group scaffolds by hash.
	groups := make(map[string][]catalog.Scaffold)
	for _, s := range scaffolds {
		h := catalog.ScaffoldHash(s.Category, s.Name)
		groups[h] = append(groups[h], s)
	}

	// Report collisions.
	for h, group := range groups {
		if len(group) > 1 {
			fmt.Printf("  [COLLISION] hash=%s: ", h)
			for i, s := range group {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%s/%s", s.Category, s.Name)
			}
			fmt.Println()
		}
	}

	// Write shard files + move ZIPs.
	for h, group := range groups {
		shardPath := filepath.Join(repoRoot, catalog.ScaffoldShardPath(h))
		shardDir := filepath.Dir(shardPath)

		// Create parent directories.
		if err := os.MkdirAll(shardDir, 0o755); err != nil {
			return fmt.Errorf("failed to create shard directory for hash %s: %w", h, err)
		}

		// Determine ZIP file names and move ZIPs, handling collisions.
		usedNames := make(map[string]bool)
		for i := range group {
			s := &group[i]
			key := s.Category + "/" + s.Name
			basename := filepath.Base(s.OriginalRef)

			// Resolve ZIP filename collision.
			zipName := basename + ".zip"
			if usedNames[zipName] {
				n := 2
				for {
					zipName = fmt.Sprintf("%s-%d.zip", basename, n)
					if !usedNames[zipName] {
						break
					}
					n++
				}
			}
			usedNames[zipName] = true

			// Set template_ref to the shard directory path.
			relShardDir := filepath.ToSlash(filepath.Dir(catalog.ScaffoldShardPath(h)))
			s.TemplateRef = relShardDir + "/" + zipName

			// Move ZIP from temp to shard directory.
			srcZip := zipPaths[key]
			dstZip := filepath.Join(shardDir, zipName)
			if err := moveFile(srcZip, dstZip); err != nil {
				return fmt.Errorf("failed to move ZIP for %s: %w", key, err)
			}

			fmt.Printf("  [ZIP] %s -> %s\n", key, dstZip)
		}

		shard := catalog.ShardFile{Scaffolds: group}
		data, err := yaml.Marshal(&shard)
		if err != nil {
			return fmt.Errorf("failed to marshal shard file for hash %s: %w", h, err)
		}

		if err := os.WriteFile(shardPath, data, 0o644); err != nil {
			return fmt.Errorf("failed to write shard file %s: %w", shardPath, err)
		}

		fmt.Printf("  [SHARD] %s -> %s (%d scaffold(s))\n", h, shardPath, len(group))
	}

	return nil
}

// moveFile moves a file from src to dst by copying and removing.
func moveFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return err
	}
	return os.Remove(src)
}

// writeMetaCatalog writes meta.yaml to the top-level directory.
func writeMetaCatalog(baseDir, version, defaultScaffold string) error {
	meta := catalog.MetaCatalog{
		Version:         version,
		DefaultScaffold: defaultScaffold,
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}

	data, err := yaml.Marshal(&meta)
	if err != nil {
		return fmt.Errorf("failed to marshal meta catalog: %w", err)
	}

	metaPath := filepath.Join(baseDir, "meta.yaml")
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write meta.yaml: %w", err)
	}

	fmt.Printf("  [META] %s generated\n", metaPath)
	return nil
}

// writeCatalogIndex writes catalog.yaml (index) to the top-level directory.
func writeCatalogIndex(baseDir string, index *catalog.CatalogIndex) error {
	data, err := yaml.Marshal(index)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog index: %w", err)
	}

	indexPath := filepath.Join(baseDir, "catalog.yaml")
	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write catalog.yaml index: %w", err)
	}

	fmt.Printf("  [INDEX] %s generated\n", indexPath)
	return nil
}
