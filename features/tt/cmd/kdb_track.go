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
	flagKdbRoot      string
	flagKdbSysID     string
	flagKdbSkelWrite bool
)

// kdbLoad is the resolved workspace + effective registry for kdb-track / kdb.
type kdbLoad struct {
	Workspace       string
	File            *registry.File
	Listed          []track.ListedEntry
	RegistryPath    string
	RegistryMissing bool
	KdbRoot         string
}

var kdbTrackCmd = &cobra.Command{
	Use:           "kdb-track",
	Short:         "Track and circulate registered knowledge DBs",
	Long:          "Read the knowledge DB registry and run cross-KDB operations (list, unregistered, status-cmds, status-all, systematize, skeleton). Output is JSON (skeleton without --write prints YAML).",
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
	kdbTrackCmd.PersistentFlags().StringVar(&flagKdbRegistry, "registry", "", "Registry file (default: KDB_REGISTRY or {workspace}/{kdb-root}/registry.yaml)")
	kdbTrackCmd.PersistentFlags().StringVar(&flagKdbRoot, "kdb-root", "", "KDB root directory relative to workspace (default: KDB_ROOT or shared/kdb)")

	listCmd := newKdbTrackSub("list", "List registered knowledge DBs", runKdbTrackList)
	unregCmd := newKdbTrackSub("unregistered", "List {kdb-root}/*/corpus directories missing from the registry", runKdbTrackUnregistered)
	statusCmdsCmd := newKdbTrackSub("status-cmds", "Print each registered status command", runKdbTrackStatusCmds)
	statusAllCmd := newKdbTrackSub("status-all", "Run status for every registered knowledge DB", runKdbTrackStatusAll)
	sysCmd := newKdbTrackSub("systematize", "Run distill/dump/merge/pack for every dirty knowledge DB", runKdbTrackSystematize)
	sysCmd.Flags().StringVar(&flagKdbSysID, "id", "", "Only systematize this knowledge DB id")

	skelCmd := &cobra.Command{
		Use:           "skeleton",
		Short:         "Print or write a registry.yaml skeleton",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE:          runKdbTrackSkeleton,
	}
	skelCmd.Flags().BoolVar(&flagKdbSkelWrite, "write", false, "Write skeleton to the resolved registry path (fails if file exists)")

	kdbTrackCmd.AddCommand(listCmd, unregCmd, statusCmdsCmd, statusAllCmd, sysCmd, skelCmd)
}

func newKdbTrackSub(use, short string, run func(cmd *cobra.Command, load kdbLoad) error) *cobra.Command {
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
				return jsonout.Fail(cmd.ErrOrStderr(), helpHint(cmd), jsonout.CodeUsage,
					fmt.Sprintf("unexpected argument %q", args[0]), nil)
			}
			load, err := loadKdbRegistry(cmd, flagKdbWorkspace, flagKdbRegistry, flagKdbRoot)
			if err != nil {
				return err
			}
			return run(cmd, load)
		},
	}
}

// helpHint returns the help command line for a cobra command, e.g. "tt kdb-track list help".
func helpHint(cmd *cobra.Command) string {
	return cmd.CommandPath() + " help"
}

func loadKdbRegistry(cmd *cobra.Command, workspaceFlag, registryFlag, kdbRootFlag string) (kdbLoad, error) {
	return loadKdbRegistryTo(cmd.ErrOrStderr(), helpHint(cmd), workspaceFlag, registryFlag, kdbRootFlag)
}

func loadKdbRegistryTo(w io.Writer, hint, workspaceFlag, registryFlag, kdbRootFlag string) (kdbLoad, error) {
	var out kdbLoad
	workspace, err := paths.ResolveWorkspace(workspaceFlag)
	if err != nil {
		return out, jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), nil)
	}
	kdbRoot := paths.ResolveKdbRoot(kdbRootFlag)
	regPath := paths.RegistryPath(workspace, registryFlag, kdbRoot)
	raw, missing, err := registry.LoadOptional(regPath)
	if err != nil {
		return out, jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), map[string]any{"registry": regPath})
	}
	if !missing {
		if err := raw.Validate(); err != nil {
			return out, jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), map[string]any{"registry": regPath})
		}
	}
	listed := track.EffectiveEntries(raw, kdbRoot)
	effective := &registry.File{Version: 1, KDBs: make([]registry.Entry, 0, len(listed))}
	if raw != nil && raw.Version >= 1 {
		effective.Version = raw.Version
	}
	for _, le := range listed {
		effective.KDBs = append(effective.KDBs, le.Entry)
	}
	out = kdbLoad{
		Workspace:       workspace,
		File:            effective,
		Listed:          listed,
		RegistryPath:    regPath,
		RegistryMissing: missing,
		KdbRoot:         kdbRoot,
	}
	return out, nil
}

func runKdbTrackList(cmd *cobra.Command, load kdbLoad) error {
	payload := map[string]any{
		"entries":  load.Listed,
		"registry": load.RegistryPath,
	}
	if load.RegistryMissing {
		payload["registry_missing"] = true
		payload["message"] = "registry file not found: " + load.RegistryPath
	}
	return jsonout.OK(cmd.OutOrStdout(), payload)
}

func runKdbTrackUnregistered(cmd *cobra.Command, load kdbLoad) error {
	r, err := track.Unregistered(load.Workspace, load.KdbRoot, load.File)
	if err != nil {
		return jsonout.Fail(cmd.ErrOrStderr(), helpHint(cmd), jsonout.CodeBusiness, err.Error(), nil)
	}
	return jsonout.OK(cmd.OutOrStdout(), map[string]any{"ids": r.IDs})
}

func runKdbTrackStatusCmds(cmd *cobra.Command, load kdbLoad) error {
	return jsonout.OK(cmd.OutOrStdout(), track.StatusCmds(load.File))
}

func runKdbTrackStatusAll(cmd *cobra.Command, load kdbLoad) error {
	statuses, anyFail := track.StatusAll(cmd.Context(), load.Workspace, load.KdbRoot, load.File)
	if anyFail {
		return jsonout.Fail(cmd.ErrOrStderr(), helpHint(cmd), jsonout.CodeBusiness,
			"one or more knowledge DB status commands failed", map[string]any{"statuses": statuses})
	}
	return jsonout.OK(cmd.OutOrStdout(), map[string]any{"statuses": statuses})
}

func runKdbTrackSystematize(cmd *cobra.Command, load kdbLoad) error {
	res, code := track.Systematize(cmd.Context(), load.Workspace, load.KdbRoot, load.File, flagKdbSysID)
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	extra := map[string]any{
		"add_candidates": res.AddCandidates,
		"entries":        res.Entries,
	}
	if len(res.Warnings) > 0 {
		extra["warnings"] = res.Warnings
	}
	switch {
	case code == jsonout.CodeUsage:
		return jsonout.Fail(errOut, helpHint(cmd), code, res.Warnings[len(res.Warnings)-1], extra)
	case code != 0:
		err := jsonout.Fail(errOut, helpHint(cmd), code, "one or more knowledge DBs failed to systematize", extra)
		writeAddCandidates(out, res.AddCandidates)
		return err
	}
	if err := jsonout.OK(out, extra); err != nil {
		return err
	}
	writeAddCandidates(out, res.AddCandidates)
	return nil
}

func runKdbTrackSkeleton(cmd *cobra.Command, args []string) error {
	if len(args) > 0 && args[0] == "help" {
		agentHelpFunc(cmd, nil)
		return nil
	}
	if len(args) > 0 {
		return jsonout.Fail(cmd.ErrOrStderr(), helpHint(cmd), jsonout.CodeUsage,
			fmt.Sprintf("unexpected argument %q", args[0]), nil)
	}
	if !flagKdbSkelWrite {
		_, err := fmt.Fprint(cmd.OutOrStdout(), track.SkeletonYAML())
		return err
	}
	workspace, err := paths.ResolveWorkspace(flagKdbWorkspace)
	if err != nil {
		return jsonout.Fail(cmd.ErrOrStderr(), helpHint(cmd), jsonout.CodeBusiness, err.Error(), nil)
	}
	kdbRoot := paths.ResolveKdbRoot(flagKdbRoot)
	regPath := paths.RegistryPath(workspace, flagKdbRegistry, kdbRoot)
	if err := track.WriteSkeleton(regPath); err != nil {
		return jsonout.Fail(cmd.ErrOrStderr(), helpHint(cmd), jsonout.CodeBusiness, err.Error(), map[string]any{"path": regPath})
	}
	return jsonout.OK(cmd.OutOrStdout(), map[string]any{"path": regPath})
}

func writeAddCandidates(w io.Writer, candidates []string) {
	for _, c := range candidates {
		fmt.Fprintf(w, "ADD_CANDIDATE: %s\n", c)
	}
}
