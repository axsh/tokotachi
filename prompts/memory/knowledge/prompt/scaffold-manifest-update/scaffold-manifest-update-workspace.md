---
id: scaffold-manifest-update-workspace
knowledge_id: scaffold-manifest-update-workspace
title: Scaffold Manifest Update Workspace
status: current
category_path: prompt/scaffold-manifest-update
created_at: 2026-11-22T10:09:50.610139Z
last_updated: 2026-11-22T10:09:50.610139Z
source_event_ids:
    - E-01M80ZVX8CEYNJVD6MC63YFFM5
    - E-01M816B9S3AHNYX0WCVJ74EQX7
    - E-01M8186VJXX388A53QNNH04XTP
---

# Scaffold Manifest Update (--update)

## Workspace layout

`tt scaffold --update [workdir]` prepares a merge workspace. Typical contents:

- `backup/` — current local `prompts/manifest` snapshot
- `latest/` — freshly extracted catalog copy
- `instruction.md` — English merge policy for the AI agent
- `doc-relations-NNN.md` — sequenced relation reports (NNN = 001, 002, ...)

`tt` does **not** call an LLM. The agent reads the workdir and applies the merge into `prompts/manifest/`.

## --force vs --update

- `--force`: destructive re-apply of scaffold content
- `--update`: preserves local edits via AI-assisted merge hints from the workdir

## Identity and similarity

- Manifest frontmatter `uid` (UUID) is the stable identity for rename-aware update relations
- Character n-gram Jaccard uses `DefaultNGramN=5`, chosen by fixture dispersion margin among n=3,4,5

## Clean path vs AI merge

- Clean path: when corresponding files match by bytes (and uid alignment), apply without AI judgment
- Otherwise: AI merges using `instruction.md` and `doc-relations-NNN.md`

## Apply conventions after AI merge

- Preserve REPO-only policy files that have no uid match in `latest` (example: project-specific `database-rules.md`, `web-security-rules.md`) even when listed as `only_backup`
- Take `latest` for additive safety guards and schema improvements when uid-aligned or new upstream-only files
- Leave the `--update` workdir in place after merge; it is evidence, not disposable cleanup target
- Catalog capability `scaffold-manifest-update` documents the procedure; `prompt-manifest-update` calls it for upstream merge
