---
id: open-questions
kind: memory
title: Open Questions
status: question
topics:
  - questions
  - open
  - unresolved
triggers:
  - encountering unclear design areas
  - proposing speculative design
depends_on:
  - current-overview
evidence:
  - docs
review:
  human_required_for:
    - question_resolution
owners:
  - architecture
last_reviewed: 2026-06-04
---

# Open Questions

Unresolved items and speculative design notes. When a question is resolved,
move the resolution to the appropriate document (decisions.md, invariants.md, etc.).

## Unresolved

### OQ-001: index.md Routing Test Automation

How should the routing test for `index.md` be automated?
Options under consideration:
- Marker words in architecture documents that scripts can grep for
- Plugin messaging system for agents to report which documents they read
- Simple link validation (all links in index.md point to existing files)

### OQ-002: Compiler Implementation Strategy

When implementing the compiler (Step 3 of the roadmap), what technology
should be used?
- Pure bash scripts (simplest, consistent with existing tooling)
- Go CLI tool (stronger typing, better error handling)
- Python script (rich YAML/JSON libraries)
