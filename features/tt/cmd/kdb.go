package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/kdb/builtin"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/delegate"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/jsonout"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/registry"
)

// kdbPipelineOps are the operations every registered KDB supports.
var kdbPipelineOps = []string{"status", "distill", "dump", "merge", "pack"}

var kdbCmd = &cobra.Command{
	Use:                "kdb <id> <operation> [args...]",
	Short:              "Run an operation on one registered knowledge DB",
	Long:               "Dispatch status/distill/dump/merge/pack to the builtin backlog implementation or to the registry command of other knowledge DBs.",
	DisableFlagParsing: true,
	SilenceUsage:       true,
	SilenceErrors:      true,
	RunE:               runKdb,
}

type kdbArgs struct {
	workspace string
	registry  string
	kdbRoot   string
	prefer    string
	help      bool
	positions []string
}

// parseKdbArgs extracts tt-level flags manually because flag parsing is
// disabled (backlog subcommands carry their own flags).
func parseKdbArgs(args []string) (kdbArgs, error) {
	var a kdbArgs
	take := func(i int, name string) (string, int, error) {
		if i+1 >= len(args) {
			return "", i, fmt.Errorf("flag %s requires a value", name)
		}
		return args[i+1], i + 1, nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			a.help = true
		case arg == "--workspace":
			v, ni, err := take(i, arg)
			if err != nil {
				return a, err
			}
			a.workspace, i = v, ni
		case strings.HasPrefix(arg, "--workspace="):
			a.workspace = strings.TrimPrefix(arg, "--workspace=")
		case arg == "--registry":
			v, ni, err := take(i, arg)
			if err != nil {
				return a, err
			}
			a.registry, i = v, ni
		case strings.HasPrefix(arg, "--registry="):
			a.registry = strings.TrimPrefix(arg, "--registry=")
		case arg == "--kdb-root":
			v, ni, err := take(i, arg)
			if err != nil {
				return a, err
			}
			a.kdbRoot, i = v, ni
		case strings.HasPrefix(arg, "--kdb-root="):
			a.kdbRoot = strings.TrimPrefix(arg, "--kdb-root=")
		case arg == "--prefer":
			v, ni, err := take(i, arg)
			if err != nil {
				return a, err
			}
			a.prefer, i = v, ni
		case strings.HasPrefix(arg, "--prefer="):
			a.prefer = strings.TrimPrefix(arg, "--prefer=")
		default:
			a.positions = append(a.positions, arg)
		}
	}
	return a, nil
}

func isPipelineOp(op string) bool {
	for _, o := range kdbPipelineOps {
		if o == op {
			return true
		}
	}
	return false
}

func runKdb(cmd *cobra.Command, args []string) error {
	errOut := cmd.ErrOrStderr()
	hint := "tt kdb help"

	a, err := parseKdbArgs(args)
	if err != nil {
		return jsonout.Fail(errOut, hint, jsonout.CodeUsage, err.Error(), nil)
	}
	// The builtin backlog owns its operations, flags and help pages.
	if len(a.positions) > 0 && a.positions[0] == builtin.BacklogID {
		return runKdbBacklog(cmd, a)
	}
	if a.help || len(a.positions) == 0 || a.positions[0] == "help" {
		agentHelpFunc(cmd, nil)
		return nil
	}

	id := a.positions[0]
	rest := a.positions[1:]
	if len(rest) == 0 {
		return jsonout.Fail(errOut, hint, jsonout.CodeUsage,
			fmt.Sprintf("missing operation for knowledge DB %q (one of: %s)", id, strings.Join(kdbPipelineOps, ", ")), nil)
	}
	op := rest[0]

	if !isPipelineOp(op) {
		return jsonout.Fail(errOut, hint, jsonout.CodeUsage,
			fmt.Sprintf("unknown operation %q for knowledge DB %q (one of: %s)", op, id, strings.Join(kdbPipelineOps, ", ")), nil)
	}

	load, err := loadKdbRegistryTo(errOut, hint, a.workspace, a.registry, a.kdbRoot)
	if err != nil {
		return err
	}
	entry, ok := load.File.ByID(id)
	if !ok {
		return jsonout.Fail(errOut, hint, jsonout.CodeUsage, fmt.Sprintf("unknown knowledge DB id %q", id), nil)
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return runKdbOp(ctx, cmd, hint, load.Workspace, a, entry, op)
}

func runKdbOp(ctx context.Context, cmd *cobra.Command, hint, workspace string, a kdbArgs, e registry.Entry, op string) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	cmdLine := map[string]string{
		"status":  e.StatusCmd,
		"distill": e.DistillCmd,
		"dump":    e.DumpCmd,
		"merge":   e.MergeCmd,
		"pack":    e.PackCmd,
	}[op]
	stdout, err := delegate.Run(ctx, workspace, cmdLine)
	if err != nil {
		return jsonout.Fail(errOut, hint, jsonout.CodeBusiness, err.Error(), map[string]any{"id": e.ID, "op": op})
	}
	if strings.TrimSpace(stdout) == "" {
		return jsonout.OK(out, map[string]any{"id": e.ID, "op": op})
	}
	_, werr := fmt.Fprint(out, ensureNewline(stdout))
	return werr
}

func ensureNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
