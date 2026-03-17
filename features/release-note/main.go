package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/axsh/tokotachi/features/release-note/internal/config"
	"github.com/axsh/tokotachi/features/release-note/internal/git"
	"github.com/axsh/tokotachi/features/release-note/internal/llm"
	"github.com/axsh/tokotachi/features/release-note/internal/scanner"
	"github.com/axsh/tokotachi/features/release-note/internal/summarizer"
	"github.com/axsh/tokotachi/features/release-note/internal/writer"
)

func main() {
	toolID := flag.String("tool-id", "", "Target tool ID (e.g. 'tt')")
	version := flag.String("version", "", "Release version (e.g. 'v1.0.0')")
	repoRoot := flag.String("repo-root", "", "Repository root path")
	configPath := flag.String("config", "", "Config file path (default: ./settings/config.yaml)")
	flag.Parse()

	if *toolID == "" || *version == "" || *repoRoot == "" {
		fmt.Fprintf(os.Stderr, "Usage: release-note --tool-id <id> --version <version> --repo-root <path>\n")
		os.Exit(1)
	}

	// Resolve config path
	if *configPath == "" {
		// Default: relative to the executable's directory
		execDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get working directory: %v", err)
		}
		*configPath = filepath.Join(execDir, "settings", "config.yaml")
	}

	if err := run(*toolID, *version, *repoRoot, *configPath); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(toolID, version, repoRoot, configPath string) error {
	ctx := context.Background()

	// Step 1: Load configuration
	log.Println("[INFO] Loading configuration...")
	cfg, apiKey, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	log.Printf("[INFO] Provider: %s, Model: %s", cfg.LLM.Provider, cfg.LLM.Model)

	// Step 2: Create LLM provider
	log.Println("[INFO] Initializing LLM provider...")
	provider, err := llm.NewProvider(cfg.LLM.Provider, apiKey, cfg.LLM.Model)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Step 3: Collect Git history
	log.Println("[INFO] Collecting Git history...")
	collector := git.NewCollector(repoRoot)

	tag, err := collector.GetLatestReleaseTag(toolID)
	if err != nil {
		log.Printf("[WARN] Failed to get latest release tag: %v", err)
		tag = ""
	}

	var sinceCommit string
	if tag != "" {
		log.Printf("[INFO] Latest release tag: %s", tag)
		sinceCommit, err = collector.GetCommitSHA(tag)
		if err != nil {
			log.Printf("[WARN] Failed to get commit SHA for tag %s: %v", tag, err)
			sinceCommit = ""
		}
	} else {
		log.Println("[INFO] No previous release found. Getting all merge commits.")
	}

	branches, err := collector.GetBranchNames(sinceCommit)
	if err != nil {
		return fmt.Errorf("failed to get branch names: %w", err)
	}
	log.Printf("[INFO] Found %d branches: %s", len(branches), strings.Join(branches, ", "))

	if len(branches) == 0 {
		log.Println("[WARN] No branches found. Generating minimal release note.")
		notesDir := filepath.Join(repoRoot, "releases", "notes")
		w := writer.New(notesDir)
		return w.Write("No changes detected since last release.", version)
	}

	// Step 4: Scan for spec folders
	log.Println("[INFO] Scanning for specification folders...")
	phasesRoot := filepath.Join(repoRoot, "prompts", "phases")
	sc := scanner.NewScanner(phasesRoot)

	specs, err := sc.FindSpecFolders(branches)
	if err != nil {
		return fmt.Errorf("failed to find spec folders: %w", err)
	}
	log.Printf("[INFO] Found %d spec folders", len(specs))

	if len(specs) == 0 {
		log.Println("[WARN] No spec folders found. Generating minimal release note.")
		notesDir := filepath.Join(repoRoot, "releases", "notes")
		w := writer.New(notesDir)
		return w.Write("Changes were made but no specification documents were found.", version)
	}

	// Step 5: Summarize each branch
	log.Println("[INFO] Summarizing branches...")
	sum := summarizer.New(provider)
	var branchSummaries []string

	for _, spec := range specs {
		log.Printf("[INFO] Processing branch: %s (%s)", spec.BranchName, spec.PhaseName)

		// Read all files in the spec folder
		var contents []string
		for _, filePath := range spec.Files {
			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("[WARN] Failed to read %s: %v", filePath, err)
				continue
			}
			contents = append(contents, fmt.Sprintf("--- File: %s ---\n%s", filepath.Base(filePath), string(data)))
		}

		if len(contents) == 0 {
			log.Printf("[WARN] No readable files for branch %s", spec.BranchName)
			continue
		}

		combined := strings.Join(contents, "\n\n")
		summary, err := sum.SummarizeBranch(ctx, spec, combined)
		if err != nil {
			log.Printf("[WARN] Failed to summarize branch %s: %v", spec.BranchName, err)
			continue
		}

		branchSummaries = append(branchSummaries, summary)
		log.Printf("[INFO] Branch %s summarized.", spec.BranchName)
	}

	if len(branchSummaries) == 0 {
		log.Println("[WARN] No summaries generated. Generating minimal release note.")
		notesDir := filepath.Join(repoRoot, "releases", "notes")
		w := writer.New(notesDir)
		return w.Write("Changes were made but could not be summarized.", version)
	}

	// Step 6: Consolidate all summaries
	log.Println("[INFO] Consolidating summaries...")
	consolidated, err := sum.Consolidate(ctx, branchSummaries)
	if err != nil {
		return fmt.Errorf("failed to consolidate summaries: %w", err)
	}

	// Step 7: Write release note
	log.Println("[INFO] Writing release notes...")
	notesDir := filepath.Join(repoRoot, "releases", "notes")
	w := writer.New(notesDir)
	if err := w.Write(consolidated, version); err != nil {
		return fmt.Errorf("failed to write release notes: %w", err)
	}

	log.Printf("[INFO] Release notes written to %s/latest.md and %s/%s.md", notesDir, notesDir, version)
	return nil
}
