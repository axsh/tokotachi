---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-agent-record-branch-package
title: "Far-Knowledge: Branch Package and Knowledge Store Internals"
description: >-
  Cross-cutting knowledge about the agent record/knowledge subsystem:
  BranchPackageInfo, Slugify, and knowledge frontmatter compatibility
  with the prompt compiler.
user_visible: false
manual_only: false
status: current
body: inline
---

# Branch Package and Knowledge Store Internals

## BranchPackageInfo

`BranchPackageInfo` is a structured type defined in `features/tt/internal/agent/types.go` that holds branch package identifiers. It is embedded in intake events as `branch_package` field to provide structured branch context.

Key function `DeriveBranchPackage()` in `features/tt/internal/agent/record/branch.go` creates this struct from `GitInfo` and a `GitExecutor`.

## Slugify

`Slugify()` function in `features/tt/internal/agent/record/branch.go` converts branch names to path-safe slugs by replacing path-unsafe characters (slashes, backslashes, colons, etc.) with dashes. It is used to generate the slug component of `BranchPackageInfo.ID`.

Key behaviors:
- Converts path-unsafe characters to dashes
- Truncates long names to keep paths manageable
- Ensures output is safe for use in file system paths

## Knowledge Frontmatter Compatibility

The `agent/knowledge` package (`KnowledgeFileMeta`) and the `prompt/memory` package (`MemoryDoc`) must use compatible frontmatter fields. Knowledge files created by `tt agent knowledge add` must pass `tt prompt compile` validation.

Any knowledge markdown file under `prompts/memory/` must have these frontmatter fields:
- `id` (required): unique identifier
- `title` (required): human-readable title
- `status` (required): one of `current`, `target`, `transitional`, `question`, `deprecated`

The `knowledge_id`, `category_path`, `source_event_ids`, `created_at`, `last_updated` fields are specific to the knowledge store and are not consumed by the prompt compiler.
