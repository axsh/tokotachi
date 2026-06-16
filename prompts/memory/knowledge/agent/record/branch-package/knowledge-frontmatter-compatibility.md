---
id: knowledge-frontmatter-compatibility
knowledge_id: knowledge-frontmatter-compatibility
title: Knowledge Frontmatter Compatibility
status: current
category_path: agent/record/branch-package
created_at: 2026-06-15T14:18:45.8764887Z
last_updated: 2026-06-15T14:18:45.8764887Z
source_event_ids:
    - E-01KV5T50FR15X3EAMJD46VN6TS
---

# Knowledge Frontmatter Compatibility

## Problem

The `agent/knowledge` package (`KnowledgeFileMeta`) and the `prompt/memory` package (`MemoryDoc`) used different frontmatter field names and required fields. Knowledge files created by `tt agent knowledge add` failed `tt prompt compile` validation.

## Resolution

Added `id` (yaml:"id") and `status` (yaml:"status") fields to `KnowledgeFileMeta` in `features/tt/internal/agent/knowledge/types.go`. The `id` field mirrors `knowledge_id`, and `status` defaults to `"current"` for new entries.

## Key Constraint

Any knowledge markdown file under `prompts/memory/` must have these frontmatter fields to pass prompt compile:
- `id` (required): unique identifier
- `title` (required): human-readable title
- `status` (required): one of `current`, `target`, `transitional`, `question`, `deprecated`

The `knowledge_id`, `category_path`, `source_event_ids`, `created_at`, `last_updated` fields are specific to the knowledge store and are not consumed by the prompt compiler.
