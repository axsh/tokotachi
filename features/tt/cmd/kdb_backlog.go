package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/clihelp"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/backlog"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/builtin"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/jsonout"
	"github.com/axsh/tokotachi/features/tt/internal/kdb/paths"
)

func init() {
	// Wire the builtin backlog so kdb-track status-all / systematize use it.
	backlog.Register()
}

// kdbBacklogOps lists every `tt kdb backlog <op>`.
var kdbBacklogOps = []string{
	"add", "list", "get", "update", "resolve",
	"status", "distill", "dump", "merge", "pack", "unpack", "restore",
	"import-md", "migrate-uuids",
}

func isBacklogOp(op string) bool {
	for _, o := range kdbBacklogOps {
		if o == op {
			return true
		}
	}
	return false
}

func isHelpToken(s string) bool { return s == "help" || s == "--help" || s == "-h" }

// backlogHelpHint returns the failure-JSON help hint for an op.
func backlogHelpHint(op string) string {
	if op == "" {
		return "tt kdb backlog help"
	}
	return "tt kdb backlog " + op + " help"
}

// printBacklogHelp renders the registered help page for "kdb backlog [op]".
func printBacklogHelp(w io.Writer, op string) {
	topic := "kdb backlog"
	if op != "" {
		topic += " " + op
	}
	if p, ok := clihelp.Lookup(topic); ok {
		fmt.Fprint(w, clihelp.Render(p))
		return
	}
	fmt.Fprintf(w, "No help page registered for `tt %s`.\n", topic)
}

// runKdbBacklog dispatches `tt kdb backlog <op> [flags]` to the backlog package.
func runKdbBacklog(cmd *cobra.Command, a kdbArgs) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	op := ""
	var opArgs []string
	if len(a.positions) > 1 {
		op = a.positions[1]
		opArgs = append(opArgs, a.positions[2:]...)
	}
	// parseKdbArgs consumed --prefer; merge needs it back.
	if a.prefer != "" {
		opArgs = append(opArgs, "--prefer", a.prefer)
	}

	if op == "" || isHelpToken(op) {
		printBacklogHelp(out, "")
		return nil
	}
	hint := backlogHelpHint(op)
	if !isBacklogOp(op) {
		return jsonout.Fail(errOut, backlogHelpHint(""), jsonout.CodeUsage,
			fmt.Sprintf("unknown backlog operation %q (one of: %s)", op, strings.Join(kdbBacklogOps, ", ")), nil)
	}
	if a.help || (len(opArgs) > 0 && isHelpToken(opArgs[0])) {
		printBacklogHelp(out, op)
		return nil
	}

	// Parse flags before touching the registry or the database.
	var flags backlog.Flags
	var positional []string
	var perr error
	switch op {
	case "add":
		flags, positional, perr = backlog.ParseFlags(backlog.AddFlags, opArgs)
	case "get":
		flags, positional, perr = backlog.ParseFlags(backlog.GetFlags, opArgs)
	case "update":
		flags, positional, perr = backlog.ParseFlags(backlog.UpdateFlags, opArgs)
	case "resolve":
		flags, positional, perr = backlog.ParseFlags(backlog.ResolveFlags, opArgs)
	case "list":
		flags, positional, perr = backlog.ParseFlags(backlog.ListFlags, opArgs)
	case "merge":
		flags, positional, perr = backlog.ParseFlags([]string{"prefer"}, opArgs)
	case "migrate-uuids":
		flags, positional, perr = backlog.ParseFlags([]string{"in", "out"}, opArgs)
	default: // status distill dump pack unpack restore import-md
		flags, positional, perr = backlog.ParseFlags(nil, opArgs)
	}
	if perr != nil {
		return backlogFail(errOut, hint, perr)
	}
	if op == "import-md" && len(positional) == 0 {
		return backlogFail(errOut, hint, &backlog.OpError{Code: jsonout.CodeUsage, Msg: "missing PATH argument"})
	}

	load, err := loadKdbRegistryTo(errOut, hint, a.workspace, a.registry, a.kdbRoot)
	if err != nil {
		return err
	}
	entry, ok := load.File.ByID(builtin.BacklogID)
	if !ok {
		return jsonout.Fail(errOut, hint, jsonout.CodeUsage, "knowledge DB id \"backlog\" is not registered", nil)
	}
	p, err := paths.ForEntry(paths.InputFromEnv(load.Workspace, a.registry, load.KdbRoot), entry)
	if err != nil {
		return jsonout.Fail(errOut, hint, jsonout.CodeBusiness, err.Error(), nil)
	}
	ops := backlog.Ops{P: p}

	var fields map[string]any
	switch op {
	case "add":
		fields, err = ops.Add(flags)
	case "get":
		fields, err = ops.Get(flags)
	case "update":
		fields, err = ops.Update(flags)
	case "resolve":
		fields, err = ops.Resolve(flags)
	case "list":
		fields, err = ops.List(flags)
	case "status":
		fields, err = ops.Status()
	case "distill":
		fields, err = ops.Distill()
	case "dump":
		fields, err = ops.Dump()
	case "merge":
		// Conflicts are not a CLI failure: exit 0 with restored=false and conflict_count>0.
		fields, err = ops.Merge(flags.Get("prefer"))
	case "pack":
		fields, err = ops.Pack()
	case "unpack":
		fields, err = ops.Unpack()
	case "restore":
		fields, err = ops.Restore()
	case "import-md":
		fields, err = ops.ImportMD(positional)
	case "migrate-uuids":
		fields, err = ops.MigrateUUIDs(flags)
	}
	if err != nil {
		return backlogFail(errOut, hint, err)
	}
	return jsonout.OK(out, fields)
}

// backlogFail converts a backlog error into the failure JSON and exit code.
func backlogFail(w io.Writer, hint string, err error) error {
	var oe *backlog.OpError
	if errors.As(err, &oe) {
		return jsonout.Fail(w, hint, oe.Code, oe.Msg, oe.Extra)
	}
	return jsonout.Fail(w, hint, jsonout.CodeBusiness, err.Error(), nil)
}
