package cmd

import "github.com/axsh/tokotachi/features/tt/internal/clihelp"

func init() {
	registerKdbHelp()
}

func registerKdbHelp() {
	exTrack := "0 success\n1 business failure (registry unreadable or at least one entry failed)\n2 usage error (unknown id, bad argument)"
	json := "Output is JSON on stdout (first line). On failure: `{\"ok\":false,\"error\":\"...\",\"help\":\"<command> help\"}`. No Python or sqlite3 CLI is required."

	clihelp.Register("kdb-track", page(
		"tt kdb-track",
		"Track and circulate every knowledge DB (KDB) listed in the registry. Replaces the former shell scripts under `scripts/kdb`.",
		"When you need to list registered KDBs, find unregistered corpora, collect status, or systematize all dirty KDBs.",
		"Do not use to edit one KDB's items; use `tt kdb <id> ...`. Do not confuse with far-knowledge (`tt agent ...`).",
		"A registry file (default `{workspace}/features/kdb/registry.yaml`). Persistent flags: `--workspace`, `--registry`.",
		"Subcommands: list, unregistered, status-cmds, status-all, systematize.\n"+json,
		exTrack,
		"`tt help kdb-track list`\n`tt help kdb-track systematize`\n`tt help kdb`",
	))
	clihelp.Register("kdb-track list", page(
		"tt kdb-track list",
		"List all registered knowledge DBs with their paths and commands.",
		"Before any cross-KDB operation, to see which ids exist.",
		"Do not use to check whether data changed; use `status-all`.",
		"Readable and valid registry file.",
		"`tt kdb-track list [--workspace DIR] [--registry FILE]`\nSuccess: `{\"ok\":true,\"entries\":[{id,module_path,corpus_path,status_cmd,...}]}`.\n"+json,
		exTrack,
		"`tt help kdb-track unregistered`\n`tt help kdb-track status-all`",
	))
	clihelp.Register("kdb-track unregistered", page(
		"tt kdb-track unregistered",
		"Report `features/*/corpus` directories that are not registered in the registry.",
		"When a new knowledge DB may have been added without registering it.",
		"Do not use to list registered KDBs; use `list`.",
		"Workspace with a `features` directory (a missing directory yields an empty result).",
		"`tt kdb-track unregistered`\nSuccess: `{\"ok\":true,\"ids\":[\"orphan\"]}`.\n"+json,
		exTrack,
		"`tt help kdb-track list`",
	))
	clihelp.Register("kdb-track status-cmds", page(
		"tt kdb-track status-cmds",
		"Print the status command line of every registered KDB without running it.",
		"When you want to inspect or run status commands yourself.",
		"Do not use to get status results; use `status-all`.",
		"Readable and valid registry file.",
		"`tt kdb-track status-cmds`\nSuccess: `{\"ok\":true,\"commands\":[{\"id\":\"...\",\"status_cmd\":\"...\"}]}`.\n"+json,
		exTrack,
		"`tt help kdb-track status-all`",
	))
	clihelp.Register("kdb-track status-all", page(
		"tt kdb-track status-all",
		"Run status for every registered KDB (builtin for backlog, registry `status_cmd` for others) and aggregate the results.",
		"To learn which KDBs are dirty before systematizing.",
		"Do not use to modify data; status is read-only.",
		"`bash` on PATH for delegated KDBs (Git Bash on Windows).",
		"`tt kdb-track status-all`\nSuccess: `{\"ok\":true,\"statuses\":[{id,ok,dirty,output}]}`. All entries are processed even when one fails; on any failure `ok` is false, `statuses` is still present, and the exit code is 1.\n"+json,
		exTrack,
		"`tt help kdb-track systematize`",
	))
	clihelp.Register("kdb-track systematize", page(
		"tt kdb-track systematize",
		"For every dirty KDB (status dirty or corpus git changes), run distill, dump, merge, then pack. Reports corpus paths to `git add`.",
		"After knowledge changes, before committing corpus updates.",
		"Do not use to commit; this command never runs `git add` or `git commit`.",
		"git and (for delegated KDBs) bash available. Optional `--id <id>` limits the run to one KDB.",
		"`tt kdb-track systematize [--id ID]`\nThe first stdout line is JSON: `{\"ok\":true,\"add_candidates\":[...],\"entries\":[...]}`. Following lines are `ADD_CANDIDATE: <corpus_path>` for each path to add. Every entry is processed; a failed entry does not stop the others.\n"+json,
		exTrack,
		"`tt help kdb-track status-all`",
	))

	clihelp.Register("kdb", page(
		"tt kdb",
		"Run one operation on one registered knowledge DB: `tt kdb <id> <operation>`.",
		"When you need status/distill/dump/merge/pack for a specific KDB.",
		"Do not use to iterate all KDBs; use `tt kdb-track`.",
		"Registry entry for `<id>`. Non-builtin KDBs need `bash` on PATH. Flags: `--workspace`, `--registry`, `--prefer` (merge).",
		"Operations: status, distill, dump, merge, pack. The builtin `backlog` runs inside tt; other ids run the registry `*_cmd` and their stdout is passed through (JSON when the command emits JSON).\n"+json,
		"0 success\n1 operation failed (or backlog not wired in this build)\n2 usage error (unknown id or operation)",
		"`tt help kdb-track`",
	))
}
