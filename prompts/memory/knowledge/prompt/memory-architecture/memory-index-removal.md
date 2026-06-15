---
id: memory-index-removal
knowledge_id: memory-index-removal
title: Memory Index Removal
status: current
category_path: prompt/memory-architecture
created_at: 2026-06-15T14:19:07.1722728Z
last_updated: 2026-06-15T14:19:07.1722728Z
source_event_ids:
    - E-01KV5N0T9H82CA5559GSAT7PZR
---

# Memory Index Removal

## What Was Removed

The `prompts/memory/index.md` file and all supporting code were deleted as part of the transition to a knowledge-based memory system.

### Deleted Components

- `memory/indexer.go` and its tests: generated the index.md file
- `CompileResult.IndexContent` field: no longer needed
- `OutputConfig.MemoryIndex` field: no longer controls index output path
- `template.go resolveRef` memory kind handler: template refs no longer resolve memory documents
- Memory confirmation steps in procedures: agents no longer confirm memory writes via index
- `deny-direct-edit-of-index` guard YAML: the guard was protecting a file that no longer exists
- `far-knowledge-memory` policy index.md references: policy rewritten to reference knowledge categories instead

## Rationale

The index.md approach was a monolithic document that grew unbounded. It has been replaced by a categorized knowledge store under `prompts/memory/knowledge/`, where each knowledge item is a separate markdown file with YAML frontmatter, organized in category directories.
