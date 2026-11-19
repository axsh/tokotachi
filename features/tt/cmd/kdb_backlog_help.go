package cmd

import "github.com/axsh/tokotachi/features/tt/internal/clihelp"

func init() {
	registerKdbBacklogHelp()
}

func registerKdbBacklogHelp() {
	const emb = "The backlog runs inside tt with an embedded SQLite engine; no sqlite3 CLI and no Python are required."
	const jsonNote = "Success: one JSON object on stdout (`{\"ok\":true,...}`). Failure: `{\"ok\":false,\"error\":\"...\",\"help\":\"tt kdb backlog <op> help\"}` on stdout."
	const wsFlags = "Common flags: `--workspace DIR`, `--registry FILE`. Environment overrides: `BL_ROOT`, `BL_DATA_DIR`, `BL_DB_PATH`, `BL_CORPUS_DIR`, `BL_TMP_DIR`, `BL_TODAY` (YYYY-MM-DD, tests only)."
	const pre = "A registry entry with id `backlog` (default `features/kdb/registry.yaml`). Help pages need neither the registry nor the database. "
	exCRUD := "0 success\n1 item not found, key exists, or database error\n2 usage error (missing/invalid flag)"
	exPipe := "0 success\n1 invalid or missing input, corrupt corpus, or database error\n2 usage error (invalid flag)"

	clihelp.Register("kdb backlog", page(
		"tt kdb backlog",
		"Manage backlog items: provisional decisions, questions to confirm, and postponed matters. Items live in a local SQLite cache and are shared through a Git corpus.",
		"When specification work produces something that is decided provisionally, needs confirmation, or is postponed; to find or update such items.",
		"Do not create Markdown files for backlog items. Do not read or write the SQLite file directly.",
		pre+emb,
		"`tt kdb backlog <op> [flags]`\nItem operations: `add`, `list`, `get`, `update`, `resolve`, `import-md`, `migrate-uuids`.\nPipeline operations: `status`, `distill`, `dump`, `merge`, `pack`, `unpack`, `restore`.\nAllowed states: `未確認`, `確認中`, `決着`, `取り下げ`. Allowed kinds: `仮の決定`, `確認事項`, `先送り`.\nReference an item in documents as `backlog:<uuid>`.\n"+wsFlags+"\n"+jsonNote,
		"0 success\n1 operation failed\n2 usage error",
		"`tt help kdb backlog add`\n`tt help kdb backlog list`\n`tt help kdb backlog status`\n`tt help kdb-track`",
	))

	clihelp.Register("kdb backlog add", page(
		"tt kdb backlog add",
		"Insert a new backlog item (provisional decision, question to confirm, or postponed matter).",
		"When specification work produces something that is decided provisionally, needs confirmation, or is postponed. Run `list` first to avoid duplicates.",
		"Do not use it to change an existing item; use `update` or `resolve`. Do not create Markdown files for backlog items.",
		pre+"The logical key (phase, branch, id) must not exist yet; otherwise exit 1. "+emb,
		"`tt kdb backlog add --phase PHASE --branch BRANCH --title TEXT --kind KIND [options]`\nRequired: `--phase`, `--branch` (directory names such as `000-foundation`), `--title` (non-empty), `--kind` (`仮の決定`, `確認事項` or `先送り`).\nOptional: `--id NNN` (omit to use the next free id), `--slug`, `--ask-to`, `--due`, `--origin`, `--problem`, `--current-handling`, `--options`, `--questions`, `--after-decision`, `--history`, `--decision`, `--state` (default `未確認`), `--state-date YYYY-MM-DD` (default today UTC).\n`created_at` and `updated_at` are today; a new random RFC 4122 `uuid` is issued once and never changes. With `--id` omitted, a concurrent clash is retried once with a fresh id.\nSuccess: `{\"ok\":true,\"item\":{...}}`. Use the `uuid` of the output when referencing the item as `backlog:<uuid>`.\n"+wsFlags,
		exCRUD,
		"`tt help kdb backlog list`\n`tt help kdb backlog get`\n`tt help kdb backlog update`\n`tt help kdb backlog resolve`",
	))

	clihelp.Register("kdb backlog list", page(
		"tt kdb backlog list",
		"List backlog items as JSON, optionally filtered, sorted and paged.",
		"Before creating a new item (use `--match`), to find open items for a phase or branch, or to count items by state or kind.",
		"Do not read the SQLite file or write SQL yourself. Use `get` for one item with a known uuid or key.",
		pre+"The local cache is current (`restore` after `unpack` on a fresh checkout). "+emb,
		"`tt kdb backlog list [--phase P] [--branch B] [--state S] [--kind K] [--match REGEX] [--include-closed 1] [--sort KEY] [--order asc|desc] [--limit N] [--offset N]`\nFilters combine with AND. `--match` is a case-insensitive regular expression searched in `title`, `problem` and `slug`. By default `決着` and `取り下げ` items are excluded unless `--state` is given or `--include-closed 1` is set.\n`--sort` is one of `local_id`, `updated_at`, `created_at`, `state`, `title` (default order: phase, branch, local_id). `--limit` >= 1, `--offset` >= 0.\nSuccess: `{\"ok\":true,\"total\":T,\"limit\":N|null,\"offset\":O,\"items\":[...]}`; `total` counts matches before limit and offset.\n"+wsFlags,
		"0 success (also when nothing matches)\n1 database error\n2 usage error (unknown flag, invalid regex, limit, offset, sort or order)",
		"`tt help kdb backlog get`\n`tt help kdb backlog add`",
	))

	clihelp.Register("kdb backlog get", page(
		"tt kdb backlog get",
		"Print one backlog item as JSON, addressed by its uuid or by its logical key.",
		"When a document references `backlog:<uuid>` and you need the full content, or when you know phase, branch and id.",
		"Do not use it to search; use `list` with filters.",
		pre+"The local cache is current. "+emb,
		"`tt kdb backlog get --uuid UUID`\n`tt kdb backlog get --phase P --branch B --id NNN`\nWhen both forms are given, the item found by uuid must have the same key; otherwise exit 1.\nSuccess: `{\"ok\":true,\"item\":{...}}` with every column of the `items` table.\n"+wsFlags,
		"0 success\n1 item not found, uuid and key mismatch, or database error\n2 usage error (no selector, incomplete key, invalid uuid)",
		"`tt help kdb backlog list`\n`tt help kdb backlog update`",
	))

	clihelp.Register("kdb backlog update", page(
		"tt kdb backlog update",
		"Update selected fields of one existing backlog item. Only the fields you pass change; `updated_at` becomes today (UTC).",
		"To correct or extend a section, to record progress with `--append-history`, or to move the state between `未確認` and `確認中`.",
		"Do not use it to settle (`決着`) or withdraw (`取り下げ`) an item; use `resolve`. Do not use it to create an item.",
		pre+"The item exists. "+emb,
		"`tt kdb backlog update --phase P --branch B --id NNN [field flags]`\nAt least one field flag is required. Field flags replace the whole column: `--slug`, `--title`, `--state`, `--state-date`, `--kind`, `--ask-to`, `--due`, `--origin`, `--decision`, `--problem`, `--current-handling`, `--options`, `--questions`, `--after-decision`, `--history`.\n`--append-history TEXT` appends TEXT on a new line (cannot be combined with `--history`). `--state` accepts only `未確認` or `確認中`. `--state-date` is YYYY-MM-DD or empty to clear. `--title` must not be empty.\nSuccess: `{\"ok\":true,\"item\":{...}}`.\n"+wsFlags,
		"0 success\n1 item not found or database error\n2 usage error (no field flag, invalid value, closing state)",
		"`tt help kdb backlog get`\n`tt help kdb backlog resolve`",
	))

	clihelp.Register("kdb backlog resolve", page(
		"tt kdb backlog resolve",
		"Close a backlog item: set its state to `決着` (settled) or `取り下げ` (withdrawn), set `state_date` and `updated_at` to today (UTC), and optionally store the decision text.",
		"When the requester has decided, or the item is no longer relevant.",
		"For partial edits or history notes, use `update`.",
		pre+"The item exists. "+emb,
		"`tt kdb backlog resolve --phase P --branch B --id NNN --state STATE [--decision TEXT]`\n`--state` is `決着` or `取り下げ`. `--decision` replaces the `decision` column when given.\nSuccess: `{\"ok\":true,\"item\":{...}}`.\n"+wsFlags,
		"0 success\n1 item not found or database error\n2 usage error (invalid state)",
		"`tt help kdb backlog update`\n`tt help kdb backlog get`",
	))

	clihelp.Register("kdb backlog status", page(
		"tt kdb backlog status",
		"Report whether the local backlog cache (SQLite) differs from the Git corpus, so you know whether distill, dump, merge and pack are needed.",
		"Before committing (if `dirty` is true run the four-step flow), after pulling, and as the status of the registry entry.",
		"It does not modify anything except creating an empty cache when none exists.",
		pre+emb,
		"`tt kdb backlog status`\nThe database hash is computed like the corpus `content_hash` (sorted items, chunks of 100, SHA-256 of the concatenated chunk hashes). `dirty` is true when the corpus manifest is missing, the hashes differ, or `data/items.json` exists and differs from the database.\nSuccess: `{\"ok\":true,\"data\":{\"dirty\":bool,\"item_count\":N,\"corpus_hash\":\"...\",\"db_hash\":\"...\",\"data_hash\":\"...\",\"reasons\":[...]}}`.\n"+wsFlags,
		"0 success (whether or not dirty)\n1 database or corpus read error\n2 usage error",
		"`tt help kdb backlog distill`\n`tt help kdb backlog dump`\n`tt help kdb backlog unpack`",
	))

	clihelp.Register("kdb backlog distill", page(
		"tt kdb backlog distill",
		"Validate the backlog before it is shared: invalid states and kinds, empty titles, empty or malformed or duplicate `uuid`, and duplicate logical keys. Identical duplicates in `data/items.json` are collapsed.",
		"Step 1 of the four-step flow (distill, dump, merge, pack), and again after `merge` when it reported conflicts.",
		"It never repairs content except collapsing identical duplicate rows in `data/items.json`.",
		pre+emb,
		"`tt kdb backlog distill`\nChecks every database row and `data/items.json` when it exists. Allowed states: `未確認`, `確認中`, `決着`, `取り下げ`. Allowed kinds: `仮の決定`, `確認事項`, `先送り`.\nSuccess: `{\"ok\":true,\"data\":{\"db\":{...report},\"data_json\":{...report}|null}}`. On problems: `{\"ok\":false,\"error\":\"distill found problems\",\"reports\":[...]}`.\n"+wsFlags,
		"0 nothing to fix (or only identical duplicates collapsed)\n1 problems found\n2 usage error",
		"`tt help kdb backlog dump`\n`tt help kdb backlog merge`\n`tt help kdb backlog update`",
	))

	clihelp.Register("kdb backlog dump", page(
		"tt kdb backlog dump",
		"Export every item of the local cache as one JSON array to `tmp/kdb/backlog/items.json` (or `$BL_TMP_DIR/items.json`).",
		"Step 2 of the four-step flow. The dump is the current-local-state input of `merge`.",
		"The dump is a transient file under `tmp/`. Do not edit or commit it.",
		pre+emb,
		"`tt kdb backlog dump`\nThe array is sorted by phase, branch, local_id and written as a single JSON line.\nSuccess: `{\"ok\":true,\"data\":{\"path\":\"...\",\"item_count\":N}}`.\n"+wsFlags,
		"0 success (an empty database writes `[]`)\n1 database or write error\n2 usage error",
		"`tt help kdb backlog merge`\n`tt help kdb backlog distill`",
	))

	clihelp.Register("kdb backlog merge", page(
		"tt kdb backlog merge",
		"Union the dump (`tmp/kdb/backlog/items.json`) and the expanded corpus (`data/items.json`) into `data/items.json`, then restore the local cache from the result. The merge never decides which duplicate is right.",
		"Step 3 of the four-step flow, after `dump` (and `unpack` when you want teammates' items).",
		"Do not use it to edit content. If both sides differ for one key, resolve it explicitly with `--prefer`.",
		pre+"`dump` has produced `items.json` in the tmp directory. "+emb,
		"`tt kdb backlog merge [--prefer tmp|data]`\nRows with the same key and identical content are kept once. Rows with the same key but different content are both kept (a conflict); the cache is not restored and `distill` fails until the conflict is resolved. `--prefer tmp` keeps the local dump version, `--prefer data` keeps the corpus version for every conflicting key.\nSuccess: `{\"ok\":true,\"data\":{\"merge\":{count,collapsed,conflict_count,conflicts,...},\"restored\":bool}}`. On conflicts the exit code is still 0: check `restored` (false) and `conflict_count` (> 0).\n"+wsFlags,
		"0 success (also when conflicts remain)\n1 missing dump or read/write error\n2 usage error (invalid flag or --prefer value)",
		"`tt help kdb backlog distill`\n`tt help kdb backlog pack`\n`tt help kdb backlog restore`",
	))

	clihelp.Register("kdb backlog pack", page(
		"tt kdb backlog pack",
		"Turn `data/items.json` into the Git-tracked corpus: sorted chunks `items/chunk-NNNN.json` (100 items each) and `manifest.json` with a `content_hash`.",
		"Step 4 (last) of the four-step flow, when the result should be committed.",
		"Do not edit corpus files by hand. Do not pack while `distill` reports problems.",
		pre+"`data/items.json` exists and has no duplicate logical keys. "+emb,
		"`tt kdb backlog pack`\n`manifest.json` has `kdb`, `format_version`, `content_hash` (SHA-256 of the concatenated chunk sha256 values), `generated_at`, `chunk_size` (100) and `chunks[]`. Stale chunk files are removed. `generated_at` is kept when the content did not change, so an unchanged pack produces no Git diff.\nSuccess: `{\"ok\":true,\"data\":{\"item_count\":N,\"chunk_count\":N,\"content_hash\":\"...\",\"corpus\":\"...\"}}`.\n"+wsFlags,
		exPipe,
		"`tt help kdb backlog unpack`\n`tt help kdb backlog status`\n`tt help kdb backlog distill`",
	))

	clihelp.Register("kdb backlog unpack", page(
		"tt kdb backlog unpack",
		"Expand the Git corpus into `data/items.json` (one JSON array, gitignored).",
		"After cloning or pulling, before `restore`, or before `merge` when you want to combine your local dump with the shared corpus.",
		"It overwrites `data/items.json` and does not touch the SQLite cache; run `restore` for that.",
		pre+"`corpus/manifest.json` exists and every chunk matches its recorded sha256, count and the manifest `content_hash`. "+emb,
		"`tt kdb backlog unpack`\nSuccess: `{\"ok\":true,\"data\":{\"item_count\":N,\"content_hash\":\"...\",\"out\":\"...\"}}`. A missing manifest, missing chunk or hash mismatch fails and leaves `data/items.json` unchanged.\n"+wsFlags,
		"0 success\n1 corpus missing or corrupt\n2 usage error",
		"`tt help kdb backlog restore`\n`tt help kdb backlog merge`\n`tt help kdb backlog status`",
	))

	clihelp.Register("kdb backlog restore", page(
		"tt kdb backlog restore",
		"Rebuild the local SQLite cache from `data/items.json`: delete every row and insert every item in one transaction.",
		"After `unpack` (fresh checkout or pull), and automatically at the end of a conflict-free `merge`.",
		"It discards all local rows that are not in `data/items.json`. Run `dump` and `merge` first if the cache has unshared changes.",
		pre+"`data/items.json` exists. "+emb,
		"`tt kdb backlog restore`\nSuccess: `{\"ok\":true,\"data\":{\"restored\":N,\"engine\":\"sqlite\"}}`. On any error the transaction is rolled back and the cache is unchanged.\n"+wsFlags,
		"0 success\n1 missing `data/items.json`, duplicate keys, or database error\n2 usage error",
		"`tt help kdb backlog unpack`\n`tt help kdb backlog status`\n`tt help kdb backlog list`",
	))

	clihelp.Register("kdb backlog import-md", page(
		"tt kdb backlog import-md",
		"Convert legacy Markdown backlog files into rows of the backlog database (migration and re-import). Existing rows with the same logical key are replaced.",
		"One-time migration of `prompts/phases/<phase>/branches/<branch>/backlogs/<NNN>-<Slug>.md`, or re-importing a corrected legacy file.",
		"Not for creating new items (use `add`). Not a sync tool: the database is the source of truth after migration.",
		pre+"Each path contains `prompts/phases/<phase>/branches/<branch>/backlogs/<NNN>-<Slug>.md`. "+emb,
		"`tt kdb backlog import-md PATH [PATH ...]`\nEach PATH is a Markdown file or a directory (searched recursively for `*.md`; files outside the backlogs layout are skipped with a warning). Header lines `- 状態:`, `- 種類:`, `- 確かめる相手:`, `- いつまでに:`, `- 出どころ:` and sections such as `## 何が問題か` map to columns; unknown values become warnings.\nSuccess: `{\"ok\":true,\"imported\":N,\"files\":N,\"warnings\":[...],\"keys\":[...]}`. Nothing is written on failure.\n"+wsFlags,
		"0 success\n1 two files map to one key, no files found, or database error\n2 usage error (path not found or outside the backlogs layout)",
		"`tt help kdb backlog list`\n`tt help kdb backlog distill`",
	))

	clihelp.Register("kdb backlog migrate-uuids", page(
		"tt kdb backlog migrate-uuids",
		"Assign a new uuid to every item of a JSON array file that lacks a valid one, and drop the legacy `logical_uuid` member.",
		"Once, when an older items array (for example a legacy `data/items.json`) has items without `uuid`.",
		"Do not use it on the SQLite cache directly; edit the JSON array, then `restore`. Existing valid uuids are never changed.",
		"A JSON array file readable by the caller (`--in -` reads stdin). Optional (this operation is not part of the main flow). "+emb,
		"`tt kdb backlog migrate-uuids --in FILE [--out FILE]`\nWithout `--out` the input file is rewritten in canonical form.\nSuccess: `{\"ok\":true,\"count\":N,\"assigned\":N,\"logical_uuid_removed\":N,\"out\":\"...\"}`.\n"+wsFlags,
		"0 success\n1 invalid JSON or write error\n2 usage error (missing `--in`, `--out -`)",
		"`tt help kdb backlog restore`\n`tt help kdb backlog distill`",
	))
}
