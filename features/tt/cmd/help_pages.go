package cmd

import "github.com/axsh/tokotachi/features/tt/internal/clihelp"

func init() {
	registerWorktreeHelp()
	registerPromptHelp()
	registerAgentHelp()
	registerMiscHelp()

	rootCmd.Short = "Tokotachi orchestrator (agent manuals: tt help <command>)"
}

func registerWorktreeHelp() {
	ex := "0 success\n1 failure"
	clihelp.Register("up", page(
		"tt up",
		"Start the container for a branch feature. The worktree must already exist.",
		"After `tt create`, when you need the Dev Container running.",
		"Do not use when you only need worktree status; use `tt status`.",
		"Docker available; worktree created with `tt create`.",
		"`tt up <feature>`\nFlags: see command. Success starts/attaches the container.",
		ex,
		"`tt help down`\n`tt help status`",
	))
	clihelp.Register("down", page(
		"tt down",
		"Stop and remove the container for the given feature.",
		"When finishing a session but keeping the worktree.",
		"Do not use to delete the branch; use `tt delete` or `tt close`.",
		"Feature argument required; container may already be stopped.",
		"`tt down <feature>`",
		ex,
		"`tt help up`\n`tt help close`",
	))
	clihelp.Register("create", page(
		"tt create",
		"Create a new git branch and set up a worktree for development.",
		"Starting work on a new feature branch.",
		"Do not recreate an existing worktree path.",
		"Clean git repository; write access to worktree parent.",
		"`tt create <branch>` (and feature options as documented by flags).",
		ex,
		"`tt help up`\n`tt help open`",
	))
	clihelp.Register("delete", page(
		"tt delete",
		"Remove worktree and delete branch. Fails if any Dev Container is still running.",
		"When the branch is finished and containers are down.",
		"Do not force-delete while containers run; stop them first.",
		"No running containers for the feature.",
		"`tt delete <branch>`",
		ex,
		"`tt help down`\n`tt help close`",
	))
	clihelp.Register("open", page(
		"tt open",
		"Syntax sugar: runs create → up → editor in sequence.",
		"When starting fresh work in one step.",
		"Do not use for attach-only; use `tt editor --attach`.",
		"Git and Docker available.",
		"`tt open <branch>`",
		ex,
		"`tt help create`\n`tt help up`\n`tt help editor`",
	))
	clihelp.Register("close", page(
		"tt close",
		"Syntax sugar: runs down → delete. Optional feature stops only that container.",
		"When ending a branch session.",
		"Do not use if you need to keep the worktree.",
		"Writable git worktree; containers stoppable.",
		"`tt close [feature]`",
		ex,
		"`tt help down`\n`tt help delete`",
	))
	clihelp.Register("editor", page(
		"tt editor",
		"Open the editor for the given branch. Use --attach to reconnect to a running container.",
		"When the environment is up and you need the IDE.",
		"Do not use before `tt up` unless attach semantics apply.",
		"Editor tooling installed; container running for attach.",
		"`tt editor <feature> [--attach]`",
		ex,
		"`tt help up`\n`tt help open`",
	))
	clihelp.Register("status", page(
		"tt status",
		"Show worktree and container status. Omitting feature shows worktree-only status.",
		"Before up/down decisions.",
		"Do not use as a substitute for doctor diagnostics.",
		"Repository checkout.",
		"`tt status [feature]`",
		ex,
		"`tt help doctor`\n`tt help list`",
	))
	clihelp.Register("list", page(
		"tt list",
		"Without arguments, list worktree branches. With a branch, list features for that branch.",
		"Discovering active worktrees/features.",
		"Do not use for container health; use `tt status` / `tt doctor`.",
		"Git worktrees readable.",
		"`tt list [branch] [--forge github|gitlab] [--update]`\n"+
			"Host resolution (highest priority first): `--forge` (this process only), then `TT_FORGE`, then `git remote get-url origin` (github/gitlab heuristics).\n"+
			"Related env: `TT_FORGE`, `TT_CMD_GH` (default gh), `TT_CMD_GLAB` (default glab). Invalid `--forge` fails the command.",
		ex,
		"`tt help status`",
	))
	clihelp.Register("shell", page(
		"tt shell",
		"Open an interactive shell in the running container.",
		"Debugging inside the Dev Container.",
		"Do not use when the container is down; run `tt up` first.",
		"Feature argument; running container.",
		"`tt shell <feature>`",
		ex,
		"`tt help exec`\n`tt help up`",
	))
	clihelp.Register("exec", page(
		"tt exec",
		"Execute a command inside the running container. Use -- to separate tt args from the command.",
		"Running one-shot commands in the container.",
		"Do not use for interactive shells; use `tt shell`.",
		"Running container; feature argument.",
		"`tt exec <feature> -- <command...>`",
		ex,
		"`tt help shell`",
	))
}

func registerPromptHelp() {
	ex := "0 success\n1 failure"
	clihelp.Register("prompt", page(
		"tt prompt",
		"Manage prompt manifest compilation and deployment (ACS via --tags). Template refs: `{{kind:id}}` (see `tt prompt instruction`).",
		"When updating agent rules/skills from the prompt manifest, or selecting an Agent Config Set with tags.",
		"Do not hand-edit deployed `.cursor`/`.claude` outputs as the source of truth.",
		"Workspace with `prompts/manifest`.",
		"Subcommands: compile, deploy, update, tags, instruction. Path flags: --workspace, --prompts-dir, …\n"+
			"ACS: `--tags` / `TT_TAGS` (default baseline). Discover tags with `tt prompt tags`. Prefer one manifest tree over A/B/C folder copies.",
		ex,
		"`tt help prompt tags`\n`tt help prompt update`\n`tt help prompt instruction`",
	))
	clihelp.Register("prompt tags", page(
		"tt prompt tags",
		"List ACS selection tags and which code_content entities belong to each tag.",
		"When discovering available Agent Config Sets before compile/deploy/update.",
		"Do not use tags list as a substitute for deploy; it is read-only discovery.",
		"Readable prompts tree with project.yaml.",
		"`tt prompt tags` (path flags as for compile).",
		ex,
		"`tt help prompt`\n`tt help prompt update`",
	))
	clihelp.Register("prompt compile", page(
		"tt prompt compile",
		"Compile prompt manifest and memory documents into the build directory.",
		"When you need a resolved manifest without deploying.",
		"Do not use alone when editors must be updated; use deploy/update.",
		"Readable prompts tree.",
		"`tt prompt compile` with path flags as needed.",
		ex,
		"`tt help prompt deploy`\n`tt help prompt update`",
	))
	clihelp.Register("prompt deploy", page(
		"tt prompt deploy",
		"Compile and deploy prompt files to target editor directories.",
		"When applying compiled prompts to a target.",
		"Do not deploy without reviewing dry-run when unsure.",
		"Writable deploy root.",
		"`tt prompt deploy --target <name>` (and path flags).",
		ex,
		"`tt help prompt update`",
	))
	clihelp.Register("prompt update", page(
		"tt prompt update",
		"Check for changes and update prompt files if needed.",
		"After editing manifest sources; typical agent workflow step.",
		"Do not skip when skills/rules must stay in sync.",
		"Git-readable prompts; writable deploy roots.",
		"`tt prompt update --target <name>`",
		ex,
		"`tt help prompt`\n`tt help prompt instruction`",
	))
	clihelp.Register("prompt instruction", page(
		"tt prompt instruction",
		"Print the target-independent `{{kind:id}}` template reference manual to stdout.",
		"When writing or reviewing manifest bodies that use template refs.",
		"Do not confuse with `tt help prompt instruction` (this Page); run the subcommand for the full manual.",
		"None (read-only).",
		"`tt prompt instruction` prints the manual (not this help Page).",
		ex,
		"`tt help prompt update`",
	))
}

func registerAgentHelp() {
	ex := "0 success\n1 failure"
	clihelp.Register("agent", page(
		"tt agent",
		"Manage agent memory, intake events, far-knowledge categories, and status.",
		"When recording or inspecting agent memory.",
		"Do not confuse with knowledge-database (`tt kdb*`); far-knowledge is a separate system.",
		"Prompts memory directories writable as needed.",
		"Subcommands: record, status, intake, knowledge.",
		ex,
		"`tt help agent record`\n`tt help agent knowledge`",
	))
	clihelp.Register("agent record", page(
		"tt agent record",
		"Record a far-knowledge intake event from a coding agent (JSON file/stdin or flags).",
		"After architecture-impacting work, before push when far-knowledge was found.",
		"Do not record discoverable local facts; only far-knowledge.",
		"Valid payload schema; memory store path writable.",
		"JSON mode or flag mode (mutually exclusive). stdout JSON result.",
		ex,
		"`tt help agent intake`\n`tt help agent knowledge`",
	))
	clihelp.Register("agent status", page(
		"tt agent status",
		"Show agent memory / intake status summaries.",
		"Before systematize or push checks.",
		"Do not use as a substitute for `tt doctor`.",
		"Readable memory index.",
		"`tt agent status`",
		ex,
		"`tt help agent intake`",
	))
	clihelp.Register("agent intake", page(
		"tt agent intake",
		"Inspect intake events (list/show/processed).",
		"When reviewing pending far-knowledge intake.",
		"Do not edit intake JSON by hand unless recovering from failure.",
		"Memory index available.",
		"Subcommands: list, show, processed.",
		ex,
		"`tt help agent intake list`\n`tt help agent record`",
	))
	clihelp.Register("agent intake list", page(
		"tt agent intake list",
		"List intake events.",
		"Scanning pending or recent intake.",
		"Do not use for knowledge category trees; use `tt agent knowledge list`.",
		"Readable index.",
		"`tt agent intake list`",
		ex,
		"`tt help agent intake show`",
	))
	clihelp.Register("agent intake show", page(
		"tt agent intake show",
		"Show one intake event.",
		"Inspecting a specific event id.",
		"Do not mutate via show.",
		"Event id exists.",
		"`tt agent intake show <id>`",
		ex,
		"`tt help agent intake list`",
	))
	clihelp.Register("agent intake processed", page(
		"tt agent intake processed",
		"Mark or list processed intake handling (see command flags).",
		"After systematize consumes events.",
		"Do not mark processed without compiling knowledge.",
		"Writable index.",
		"`tt agent intake processed` (flags per command).",
		ex,
		"`tt help agent intake`",
	))
	clihelp.Register("agent knowledge", page(
		"tt agent knowledge",
		"Manage far-knowledge categories (add/append/list/split/merge/rename/move).",
		"When organizing `prompts/memory/knowledge`.",
		"Do not use for backlog/knowledge-database items; use `tt kdb backlog`.",
		"Knowledge store paths writable.",
		"Subcommands: add, append, list, split, merge, rename, move.",
		ex,
		"`tt help agent knowledge list`\n`tt help agent record`",
	))
	clihelp.Register("agent knowledge add", page(
		"tt agent knowledge add",
		"Create a new category and add a knowledge file.",
		"First note in a new category path.",
		"Do not add to an existing category; use append.",
		"Content file and category path flags.",
		"`tt agent knowledge add --category-path … --title … --content-file …`",
		ex,
		"`tt help agent knowledge append`",
	))
	clihelp.Register("agent knowledge append", page(
		"tt agent knowledge append",
		"Add a knowledge file to an existing category.",
		"Extending an existing category.",
		"Do not create a new category here; use add.",
		"Category exists.",
		"`tt agent knowledge append --category-path … --title … --content-file …`",
		ex,
		"`tt help agent knowledge list`",
	))
	clihelp.Register("agent knowledge list", page(
		"tt agent knowledge list",
		"List knowledge categories.",
		"Before add/split/merge decisions.",
		"Do not list backlog items here.",
		"Readable knowledge store.",
		"`tt agent knowledge list`",
		ex,
		"`tt help agent knowledge`",
	))
	clihelp.Register("agent knowledge split", page(
		"tt agent knowledge split",
		"Split a category into subcategories.",
		"When a category grew too broad.",
		"Do not split without a plan file when required by flags.",
		"Category path argument.",
		"`tt agent knowledge split <category-path> …`",
		ex,
		"`tt help agent knowledge merge`",
	))
	clihelp.Register("agent knowledge merge", page(
		"tt agent knowledge merge",
		"Merge categories into one.",
		"When consolidating overlapping categories.",
		"Do not merge unrelated domains.",
		"At least two category paths.",
		"`tt agent knowledge merge <path1> <path2> …`",
		ex,
		"`tt help agent knowledge split`",
	))
	clihelp.Register("agent knowledge rename", page(
		"tt agent knowledge rename",
		"Rename a category path.",
		"Fixing naming without content changes.",
		"Do not rename across unrelated trees casually.",
		"Old and new paths.",
		"`tt agent knowledge rename <old> <new>`",
		ex,
		"`tt help agent knowledge move`",
	))
	clihelp.Register("agent knowledge move", page(
		"tt agent knowledge move",
		"Move knowledge entries between categories (see flags).",
		"Reorganizing notes.",
		"Do not move without verifying references.",
		"Valid source/destination.",
		"`tt agent knowledge move` with flags.",
		ex,
		"`tt help agent knowledge`",
	))
}

func registerMiscHelp() {
	ex := "0 success\n1 failure"
	clihelp.Register("version", page(
		"tt version",
		"Print the embedded tt product version string.",
		"When confirming which release binary or local build is running.",
		"Do not use to bump the release; use the tool release procedure.",
		"None (no network, no workspace writes).",
		"`tt version`\nAlso: `tt --version` or `tt -V`\nSuccess prints `tt vN.N.N` on stdout.",
		"0 success",
		"`tt help`\n`tt help doctor`",
	))
	clihelp.Register("merge", page(
		"tt merge",
		"Local git merge of the specified branch into its base branch.",
		"Integrating a feature branch locally.",
		"Do not use instead of hosting-provider PR merge when that is required.",
		"Clean merge possible; git repository.",
		"`tt merge <branch>` (see flags for ff-only etc.).",
		ex,
		"`tt help pr`",
	))
	clihelp.Register("pr", page(
		"tt pr",
		"Create or manage pull requests for the feature branch (hosting CLI).",
		"When opening a PR after local work.",
		"Do not invent remote URLs; use configured remotes.",
		"Authenticated gh/host CLI as required.",
		"`tt pr [--forge github|gitlab] <branch> [feature]`\n"+
			"Host resolution (highest priority first): `--forge` (this process only), then `TT_FORGE`, then `git remote get-url origin` (github/gitlab heuristics).\n"+
			"Related env: `TT_FORGE`, `TT_CMD_GH` (default gh), `TT_CMD_GLAB` (default glab). Invalid `--forge` fails the command.",
		ex,
		"`tt help merge`",
	))
	clihelp.Register("doctor", page(
		"tt doctor",
		"Diagnose repository structure, configuration files, and external tool availability.",
		"When setup looks wrong or onboarding.",
		"Do not treat doctor as a substitute for `tt status`.",
		"Readable repo root.",
		"`tt doctor [--forge github|gitlab] [--json]`\n"+
			"Host resolution (highest priority first): `--forge` (this process only), then `TT_FORGE`, then `git remote get-url origin` (github/gitlab heuristics).\n"+
			"Related env: `TT_FORGE`, `TT_CMD_GH` (default gh), `TT_CMD_GLAB` (default glab). Invalid `--forge` fails the command.",
		ex,
		"`tt help status`",
	))
	clihelp.Register("scaffold", page(
		"tt scaffold",
		"Scaffold directory structures and files from predefined templates downloaded from an external repository.",
		"Bootstrapping new feature layouts, or preparing a prompts/manifest update workspace with --update.",
		"Do not overwrite existing projects without review. Prefer --update over --force when merging upstream manifests.",
		"Network access if templates are remote.",
		"`tt scaffold` with template flags.\n"+
			"`tt scaffold <category> <name> --update [workdir]` prepares backup/latest/instruction.md/doc-relations-NNN.md for AI merge.",
		ex,
		"`tt help create`",
	))
	clihelp.Register("_update-code-status", page(
		"tt _update-code-status",
		"Update code status metadata used by Tokotachi workflows.",
		"When status files must reflect current code state.",
		"Do not hand-edit generated status if this command owns it.",
		"Writable status paths.",
		"`tt _update-code-status`",
		ex,
		"`tt help doctor`",
	))
}
