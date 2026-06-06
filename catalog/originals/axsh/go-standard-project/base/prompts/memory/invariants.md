---
id: invariants
kind: memory
title: Invariants
status: current
topics:
  - invariants
  - constraints
  - rules
triggers:
  - making design changes
  - changing module boundaries
  - modifying build or test infrastructure
depends_on:
  - current-overview
evidence:
  - code
  - docs
review:
  human_required_for:
    - invariant_addition
    - invariant_removal
    - invariant_modification
owners:
  - architecture
last_reviewed: 2026-06-04
---

# Invariants

Properties that must never be violated. These are extracted from established project conventions.
It is recommended to organize them into sections and categorize them as needed.

## Workspace Boundary

- **INV-001**: No file reads, modifications, or creations outside the project root directory.
- **INV-002**: Always treat the current working directory as the absolute project root.
  Never traverse to parent directories to find or edit files.

(Empty - add new invariants below this line)