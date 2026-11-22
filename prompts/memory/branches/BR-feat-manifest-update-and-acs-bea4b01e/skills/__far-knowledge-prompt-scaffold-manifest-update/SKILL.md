---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-prompt-scaffold-manifest-update
title: "Far-Knowledge: Scaffold Manifest Update Workspace"
description: >-
  Cross-cutting knowledge about tt scaffold --update workdir layout, uid
  identity, n-gram doc-relations, and AI merge apply conventions for
  prompts/manifest.
user_visible: false
manual_only: false
status: current
body: inline
---

# Scaffold Manifest Update Workspace

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

- Preserve REPO-only policy files that have no uid match in `latest` even when listed as `only_backup`
- Take `latest` for additive safety guards and schema improvements when uid-aligned or new upstream-only files
- Leave the `--update` workdir in place after merge; it is evidence, not a disposable cleanup target
- Catalog capability `scaffold-manifest-update` documents the procedure; `prompt-manifest-update` calls it for upstream merge

## Ask User for Scaffold Update Arguments

Agents must not hardcode `tt scaffold` category/name. Ask the user for category, name, workdir, and optional flags before running `--update`, and use their answers literally.
