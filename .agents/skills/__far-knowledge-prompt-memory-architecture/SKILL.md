---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-prompt-memory-architecture
title: "Far-Knowledge: Memory System Architecture"
description: >-
  Cross-cutting knowledge about the memory system architecture:
  the removal of index.md and transition to categorized knowledge store.
user_visible: false
manual_only: false
status: current
body: inline
---

# Memory System Architecture

## Memory Index Removal

The `prompts/memory/index.md` file and all supporting code were deleted as part of the transition to a knowledge-based memory system.

### Deleted Components

- `memory/indexer.go` and its tests: generated the index.md file
- `CompileResult.IndexContent` field: no longer needed
- `OutputConfig.MemoryIndex` field: no longer controls index output path
- `template.go resolveRef` memory kind handler: template refs no longer resolve memory documents
- Memory confirmation steps in procedures: agents no longer confirm memory writes via index
- `deny-direct-edit-of-index` guard YAML: the guard was protecting a file that no longer exists
- `far-knowledge-memory` policy index.md references: policy rewritten to reference knowledge categories instead

### Current Architecture

The index.md approach was a monolithic document that grew unbounded. It has been replaced by a categorized knowledge store under `prompts/memory/knowledge/`, where each knowledge item is a separate markdown file with YAML frontmatter, organized in category directories.
