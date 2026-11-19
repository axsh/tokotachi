package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/clihelp"
)

func init() {
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command path...]",
		Short: "Show agent-oriented Markdown help (see also: tt help <command>)",
		Long:  "Print agent-oriented Markdown manuals. Prefer this over embedding usage text in skills.",
		RunE:  runAgentHelp,
	})
	rootCmd.SetHelpFunc(agentHelpFunc)
}

func runAgentHelp(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		writeRootHelp(cmd.OutOrStdout())
		return nil
	}
	target, remaining, err := rootCmd.Find(args)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Unknown help topic: %s\n\nRun `tt help` for a command list.\n", strings.Join(args, " "))
		return err
	}
	// Topics below commands that parse their own arguments (e.g. `kdb backlog add`)
	// are not cobra subcommands; resolve them against the registered pages.
	if len(remaining) > 0 {
		topic := strings.TrimSpace(commandPathOf(target) + " " + strings.Join(remaining, " "))
		if p, ok := clihelp.Lookup(topic); ok {
			fmt.Fprint(cmd.OutOrStdout(), clihelp.Render(p))
			return nil
		}
	}
	agentHelpFunc(target, nil)
	return nil
}

func agentHelpFunc(cmd *cobra.Command, _ []string) {
	out := cmd.OutOrStdout()
	path := commandPathOf(cmd)
	if path == "" {
		writeRootHelp(out)
		return
	}
	if p, ok := clihelp.Lookup(path); ok {
		fmt.Fprint(out, clihelp.Render(p))
		return
	}
	fmt.Fprint(out, clihelp.Render(synthesizeFromCobra(cmd)))
}

func commandPathOf(cmd *cobra.Command) string {
	full := strings.TrimSpace(cmd.CommandPath())
	parts := strings.Fields(full)
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[1:], " ")
}

func writeRootHelp(w io.Writer) {
	var b strings.Builder
	b.WriteString("# tt\n\n")
	b.WriteString("## Purpose\n\n")
	b.WriteString("Tokotachi (`tt`) orchestrates parallel development with AI agents: worktrees, containers, prompts, and knowledge tools.\n\n")
	b.WriteString("## When to use\n\n")
	b.WriteString("Use `tt` as the single entrypoint for environment lifecycle and agent-facing utilities.\n\n")
	b.WriteString("## When not to use\n\n")
	b.WriteString("Do not bypass `tt` with ad-hoc docker/git scripts when an equivalent `tt` command exists.\n\n")
	b.WriteString("## Preconditions\n\n")
	b.WriteString("A Tokotachi repository checkout. Some commands need Docker or git.\n\n")
	b.WriteString("## Input and output\n\n")
	b.WriteString("Agent manuals are Markdown via `tt help <command>`.\n\n")
	b.WriteString("### Commands\n\n")
	for _, c := range rootCmd.Commands() {
		if !c.IsAvailableCommand() || c.Hidden {
			continue
		}
		if c.Name() == "help" {
			continue
		}
		b.WriteString(fmt.Sprintf("- `%s` — %s\n", c.Name(), c.Short))
	}
	b.WriteString("\n## Exit codes\n\n")
	b.WriteString("0 success\n1 failure\n\n")
	b.WriteString("## Read next\n\n")
	b.WriteString("`tt help up`\n`tt help prompt`\n`tt help agent`\n")
	fmt.Fprint(w, b.String())
}

func synthesizeFromCobra(cmd *cobra.Command) clihelp.Page {
	purpose := strings.TrimSpace(cmd.Long)
	if purpose == "" {
		purpose = strings.TrimSpace(cmd.Short)
	}
	if purpose == "" {
		purpose = "No long description registered; see flags via the command itself."
	}
	title := "tt"
	if p := commandPathOf(cmd); p != "" {
		title = "tt " + p
	}
	return clihelp.Page{
		Title:          title,
		Purpose:        purpose,
		WhenToUse:      "When you need this command's documented behavior.",
		WhenNotToUse:   "When a more specific `tt` command covers the task.",
		Preconditions:  "Depends on the command; see Input and output.",
		InputAndOutput: "See command flags. Prefer registering a full clihelp.Page for this path.",
		ExitCodes:      "0 success\n1 failure",
		ReadNext:       "`tt help`",
	}
}

// page is a short helper for registering help pages.
func page(title, purpose, when, whenNot, pre, io, exits, next string) clihelp.Page {
	return clihelp.Page{
		Title:          title,
		Purpose:        purpose,
		WhenToUse:      when,
		WhenNotToUse:   whenNot,
		Preconditions:  pre,
		InputAndOutput: io,
		ExitCodes:      exits,
		ReadNext:       next,
	}
}
