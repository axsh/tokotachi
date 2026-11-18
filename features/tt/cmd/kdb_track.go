package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/jsonout"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/registry"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/track"
)

var (
	flagKdbWorkspace string
	flagKdbRegistry  string
	flagKdbSysID     string
)

var kdbTrackCmd = &cobra.Command{
	Use:           "kdb-track",
	Short:         "Track and circulate registered knowledge DBs",
	Long:          "Read the knowledge DB registry and run cross-KDB operations (list, unregistered, status-cmds, status-all, systematize). Output is JSON.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Args:          cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		agentHelpFunc(cmd, nil)
		return nil
	},
}

func init() {
	kdbTrackCmd.PersistentFlags().StringVar(&flagKdbWorkspace, "workspace", "", "Workspace root (default: TT_WORKSPACE, BL_ROOT, or cwd)")
	kdbTrackCmd.PersistentFlags().StringVar(&flagKdbRegistry, "registry", "", "Registry file (default: KDB_REGISTRY or {workspace}/features/kdb/registry.yaml)")

	listCmd := newKdbTrackSub("list", "List registered knowledge DBs", runKdbTrackList)
	unregCmd := newKdbTrackSub("unregistered", "List features/*/corpus directories missing from the registry", runKdbTrackUnregistered)
	statusCmdsCmd := newKdbTrackSub("status-cmds", "Print each registered status command", runKdbTrackStatusCmds)
	statusAllCmd := newKdbTrackSub("status-all", "Run status for every registered knowledge DB", runKdbTrackStatusAll)
	sysCmd := newKdbTrackSub("systematize", "Run distill/dump/merge/pack for every dirty knowledge DB", runKdbTrackSystematize)
	sysCmd.Flags().StringVar(&flagKdbSysID, "id", "", "Only systematize this knowledge DB id")

	kdbTrackCmd.AddCommand(listCmd, unregCmd, statusCmdsCmd, statusAllCmd, sysCmd)
}

func newKdbTrackSub(use, short string, run func(cmd *cobra.Command, workspace string, f *registry.File) error) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "help" {
				agentHelpFunc(cmd, nil)
				return nil
			}
			if len(args) > 0 {
				return jsonout.Fail(cmd.OutOrStdout(), helpHint(cmd), jsonout.CodeUsage,
					fmt.Sprintf("unexpected argument %q", args[0]), nil)
			}
			workspace, f, err := loadKdbRegistry(cmd, flagKdbWorkspace, flagKdbRegistry)
			if err != nil {
				return err
			}
			return run(cmd, workspace, f)
		},
	}
}

// helpHint returns the help command line for a cobra command, e.g. "tt kdb-track list help".
func helpHint(cmd *cobra.Command) string {
	return cmd.CommandPath() + " help"
}

// loadKdbRegistry resolves the workspace and registry, loads and validates it.
// On failure it writes a JSON error to stdout and returns an *jsonout.ExitError.
func loadKdbRegistry(cmd *cobra.Command, workspaceFlag, registryFlag string) (string, *registry.File, error) {
	return loadKdbRegistryTo(cmd.OutOrStdout(), helpHint(cmd), workspaceFlag, registryFlag)
}

func loadKdbRegistryTo(w io.Writer, hint, workspaceFlag, registryFlag string) (string, *registry.File, error) {
	workspace, err := paths.ResolveWorkspace(workspaceFlag)
	if err != nil {
		return "", nil, jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), nil)
	}
	regPath := paths.RegistryPath(workspace, registryFlag)
	f, err := registry.Load(regPath)
	if err != nil {
		return "", nil, jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), map[string]any{"registry": regPath})
	}
	if err := f.Validate(); err != nil {
		return "", nil, jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), map[string]any{"registry": regPath})
	}
	return workspace, f, nil
}

func runKdbTrackList(cmd *cobra.Command, _ string, f *registry.File) error {
	r := track.List(f)
	return jsonout.OK(cmd.OutOrStdout(), map[string]any{"entries": r.Entries})
}

func runKdbTrackUnregistered(cmd *cobra.Command, workspace string, f *registry.File) error {
	r, err := track.Unregistered(workspace, f)
	if err != nil {
		return jsonout.Fail(cmd.OutOrStdout(), helpHint(cmd), jsonout.CodeBusiness, err.Error(), nil)
	}
	return jsonout.OK(cmd.OutOrStdout(), map[string]any{"ids": r.IDs})
}

func runKdbTrackStatusCmds(cmd *cobra.Command, _ string, f *registry.File) error {
	return jsonout.OK(cmd.OutOrStdout(), track.StatusCmds(f))
}

func runKdbTrackStatusAll(cmd *cobra.Command, workspace string, f *registry.File) error {
	statuses, anyFail := track.StatusAll(cmd.Context(), workspace, f)
	if anyFail {
		return jsonout.Fail(cmd.OutOrStdout(), helpHint(cmd), jsonout.CodeBusiness,
			"one or more knowledge DB status commands failed", map[string]any{"statuses": statuses})
	}
	return jsonout.OK(cmd.OutOrStdout(), map[string]any{"statuses": statuses})
}

func runKdbTrackSystematize(cmd *cobra.Command, workspace string, f *registry.File) error {
	res, code := track.Systematize(cmd.Context(), workspace, f, flagKdbSysID)
	out := cmd.OutOrStdout()
	extra := map[string]any{
		"add_candidates": res.AddCandidates,
		"entries":        res.Entries,
	}
	if len(res.Warnings) > 0 {
		extra["warnings"] = res.Warnings
	}
	switch {
	case code == jsonout.CodeUsage:
		return jsonout.Fail(out, helpHint(cmd), code, res.Warnings[len(res.Warnings)-1], extra)
	case code != 0:
		// First line is the JSON document; ADD_CANDIDATE lines follow.
		err := jsonout.Fail(out, helpHint(cmd), code, "one or more knowledge DBs failed to systematize", extra)
		writeAddCandidates(out, res.AddCandidates)
		return err
	}
	if err := jsonout.OK(out, extra); err != nil {
		return err
	}
	writeAddCandidates(out, res.AddCandidates)
	return nil
}

func writeAddCandidates(w io.Writer, candidates []string) {
	for _, c := range candidates {
		fmt.Fprintf(w, "ADD_CANDIDATE: %s\n", c)
	}
}
