package cmd

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/prompt/compiler"
	"github.com/axsh/tokotachi/features/tt/internal/prompt/manifest"
)

var (
	tagsListProject string
)

var promptTagsListCmd = &cobra.Command{
	Use:   "tags",
	Short: "List ACS selection tags and member entities",
	Long: `List selection tags found on code_content manifests (Agent Config Set discovery).

Entities without an explicit tags field are treated as having the implicit baseline tag.
Non-taggable kinds (guard/worker/bundle/target) and memory docs are omitted.`,
	Args: cobra.NoArgs,
	RunE: runPromptTagsList,
}

func init() {
	addPromptPathFlags(promptTagsListCmd)
	promptTagsListCmd.Flags().StringVar(&tagsListProject, "project",
		"prompts/manifest/project.yaml", helpProject)
	promptCmd.AddCommand(promptTagsListCmd)
	promptCmd.Short = "Manage prompt manifests (ACS via --tags); see 'tt prompt tags' and 'tt prompt instruction' for {{kind:id}} template refs"
	promptCmd.Long = `Manage prompt manifest compilation and deployment (including ACS tag selection).

Agent Config Set (ACS): select a subset of prompts with --tags / TT_TAGS (default: baseline).
Discover tags with:  tt prompt tags
Prefer one manifest tree with tags over copying A/B/C folder sets.

Manifest bodies may use target-independent template references of the form {{kind:id}}
(policy, procedure, capability, target). They are resolved to agent-specific paths at emit time.
For the full template reference manual, run:  tt prompt instruction`
}

func runPromptTagsList(cmd *cobra.Command, args []string) error {
	pathOpts, err := buildPathOptions(cmd, tagsListProject)
	if err != nil {
		return err
	}
	paths, err := compiler.ResolvePaths(pathOpts)
	if err != nil {
		return err
	}
	cfg, err := compiler.LoadConfig(paths.ProjectYAML)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	entities, parseErrors := manifest.ParseAllEntities(cfg, paths.Workspace)
	for _, e := range parseErrors {
		fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: %s\n", e.Error())
	}

	type member struct {
		kind string
		id   string
		path string
	}
	byTag := map[string][]member{}
	for _, e := range entities {
		if !manifest.IsTaggableKind(e.Kind) {
			continue
		}
		tags, err := manifest.EffectiveTags(e)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: %s: %v\n", e.ID, err)
			continue
		}
		rel := e.FilePath
		if paths.Workspace != "" {
			if r, err := filepath.Rel(paths.Workspace, e.FilePath); err == nil {
				rel = filepath.ToSlash(r)
			}
		}
		for _, tag := range tags {
			byTag[tag] = append(byTag[tag], member{kind: e.Kind, id: e.ID, path: rel})
		}
	}

	tagNames := make([]string, 0, len(byTag))
	for t := range byTag {
		tagNames = append(tagNames, t)
	}
	sort.Strings(tagNames)

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "ACS selection tags (implicit baseline when frontmatter omits tags):")
	if len(tagNames) == 0 {
		fmt.Fprintln(out, "  (none)")
		return nil
	}
	for _, t := range tagNames {
		fmt.Fprintf(out, "- %s\n", t)
	}
	fmt.Fprintln(out)
	for _, t := range tagNames {
		fmt.Fprintf(out, "## tag: %s\n", t)
		mems := byTag[t]
		sort.Slice(mems, func(i, j int) bool {
			if mems[i].kind != mems[j].kind {
				return mems[i].kind < mems[j].kind
			}
			return mems[i].id < mems[j].id
		})
		for _, m := range mems {
			fmt.Fprintf(out, "  - %s %s (%s)\n", m.kind, m.id, m.path)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "Compile/deploy with: tt prompt compile|deploy|update --tags <a,b,...>")
	return nil
}
