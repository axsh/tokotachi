package cmd

import "github.com/axsh/tokotachi/features/tt/internal/clihelp"

func init() {
	registerKdbHelp()
}

func registerKdbHelp() {
	exTrack := "0 success\n1 business failure (registry unreadable or at least one entry failed)\n2 usage error (unknown id, bad argument)"
	json := "Output is JSON on stdout (first line). On failure: `{\"ok\":false,\"error\":\"...\",\"help\":\"<command> help\"}`. No Python or sqlite3 CLI is required."
	wsFlags := "Persistent flags: `--workspace`, `--registry`, `--kdb-root` (default `shared/kdb`; env `KDB_ROOT`)."

	clihelp.Register("kdb-track", page(
		"tt kdb-track",
		"Track and circulate every knowledge DB (KDB) listed in the registry, plus the built-in backlog. Replaces the former shell scripts under `scripts/kdb`.",
		"When you need to list registered KDBs, find unregistered corpora, collect status, or systematize all dirty KDBs.",
		"Do not use to edit one KDB's items; use `tt kdb <id> ...`. Do not confuse with far-knowledge (`tt agent ...`).",
		"Optional registry file (default `{workspace}/shared/kdb/registry.yaml`). Missing registry is OK: yaml entries are empty and built-in backlog remains. "+wsFlags,
		"Subcommands: list, unregistered, status-cmds, status-all, systematize, skeleton.\n"+
			"`*_cmd` fields in the registry are full shell command lines and may include arguments/flags.\n"+json,
		exTrack,
		"`tt help kdb-track list`\n`tt help kdb-track skeleton`\n`tt help kdb-track systematize`\n`tt help kdb`",
	))
	clihelp.Register("kdb-track list", page(
		"tt kdb-track list",
		"List all registered knowledge DBs with their paths, commands, and `source` (`builtin` or `registry`).",
		"Before any cross-KDB operation, to see which ids exist.",
		"Do not use to check whether data changed; use `status-all`.",
		"Workspace resolvable. Registry file may be missing.",
		"`tt kdb-track list [--workspace DIR] [--registry FILE] [--kdb-root DIR]`\n"+
			"Success: `{\"ok\":true,\"registry\":\"...\",\"entries\":[{id,module_path,corpus_path,status_cmd,...,source}]}`.\n"+
			"When the registry file is absent: `registry_missing:true` and `message` explaining that, exit 0; entries still include built-in backlog with `source\":\"builtin\"`.\n"+json,
		exTrack,
		"`tt help kdb-track unregistered`\n`tt help kdb-track status-all`",
	))
	clihelp.Register("kdb-track unregistered", page(
		"tt kdb-track unregistered",
		"Report `{kdb-root}/*/corpus` directories that are not registered in the effective registry.",
		"When a new knowledge DB may have been added without registering it.",
		"Do not use to list registered KDBs; use `list`. Does not scan `features/*/corpus`.",
		"Workspace with a kdb-root directory (a missing directory yields an empty result).",
		"`tt kdb-track unregistered`\nSuccess: `{\"ok\":true,\"ids\":[\"orphan\"]}`.\n"+json,
		exTrack,
		"`tt help kdb-track list`",
	))
	clihelp.Register("kdb-track status-cmds", page(
		"tt kdb-track status-cmds",
		"Print the status command line of every registered KDB without running it.",
		"When you want to inspect or run status commands yourself.",
		"Do not use to get status results; use `status-all`.",
		"Workspace resolvable. Registry may be missing (built-in backlog still listed).",
		"`tt kdb-track status-cmds`\nSuccess: `{\"ok\":true,\"commands\":[{\"id\":\"...\",\"status_cmd\":\"...\"}]}`.\n"+
			"`status_cmd` may include arguments/flags.\n"+json,
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
	clihelp.Register("kdb-track skeleton", page(
		"tt kdb-track skeleton",
		"Print a registry.yaml skeleton to stdout, or write it with `--write`.",
		"When starting a new workspace knowledge-DB registry under `shared/kdb/`.",
		"Do not use `--write` to overwrite an existing registry; it refuses if the file exists.",
		"No registry or database required. "+wsFlags,
		"`tt kdb-track skeleton` prints YAML (version 1, example `*_cmd` lines with arguments, backlog optional as comments).\n"+
			"`tt kdb-track skeleton --write` creates the resolved registry path; success JSON: `{\"ok\":true,\"path\":\"...\"}`.\n"+json,
		"0 success\n1 write/business failure (file exists or I/O error)\n2 usage error",
		"`tt help kdb-track list`",
	))

	clihelp.Register("kdb", page(
		"tt kdb",
		"Run one operation on one registered knowledge DB: `tt kdb <id> <operation>`.",
		"When you need status/distill/dump/merge/pack for a specific KDB.",
		"Do not use to iterate all KDBs; use `tt kdb-track`.",
		"Registry entry for `<id>` (built-in backlog is always available even without a registry file). Non-builtin KDBs need `bash` on PATH. Flags: `--workspace`, `--registry`, `--kdb-root`, `--prefer` (merge).",
		"Operations: status, distill, dump, merge, pack. The builtin `backlog` runs inside tt and has more operations (`tt kdb backlog <op>`, see `tt help kdb backlog`); other ids run the registry `*_cmd` (full command lines with arguments allowed) and their stdout is passed through (JSON when the command emits JSON).\n"+json,
		"0 success\n1 operation failed\n2 usage error (unknown id or operation)",
		"`tt help kdb backlog`\n`tt help kdb-track`",
	))
}
