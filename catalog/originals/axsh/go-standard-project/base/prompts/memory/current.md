---
id: current-overview
kind: memory
title: Current Project Overview
status: current
topics:
  - overview
  - modules
  - structure
triggers:
  - onboarding to the project
  - understanding the overall architecture
  - changing package boundaries
depends_on: []
evidence:
  - code
  - docs
review:
  human_required_for:
    - major_structural_change
owners:
  - architecture
last_reviewed: 2026-06-04
---

# Current Project Overview

This document describes the current state of the project structure, module responsibilities,
and dependency relationships.

## Technology Stack

- **Build System**: Custom bash scripts (`scripts/process/`)

## Repository Structure

```
.
├── features/                   # Feature modules (vertical slices)
│
├── shared/                     # Shared libraries and utilities
│   └── libs/                   # Reusable library packages
│       └── README.md
│
├── scripts/                    # Build and utility scripts
│   ├── process/                # Build pipeline scripts
│   │   ├── build.sh            # Full build + unit test runner
│   │   └── integration_test.sh # Integration test runner
│   └── utils/                  # Utility scripts
│       └── show_current_status.sh
│
├── prompts/                    # Coding agent configuration (source of truth)
│   ├── memory/              # Project Memory (this directory)
│   ├── manifest/               # Common IR manifest definitions
│   └── phases/                 # Phased development specifications
│       └── 000-foundation/
│
├── AGENTS.md                   # Workspace-level rules (sandbox boundary)
├── .gitignore
└── .gitmodules                 # Git submodule references
```

## Module Responsibilities

### features/*
Vertical slices of the project.

### shared/libs/
Reusable library packages shared across feature modules. Follows the "Accept Interfaces, Return Structs" principle. Dependencies flow inward (features depend on shared, not vice versa).

### scripts/
Build and utility scripts that automate development workflows.
- `process/`: Pipeline scripts (build, test, integration test)
- `prompt/`: Prompt manifest management scripts
- `utils/`: Helper scripts (status display, validation)

> [!IMPORTANT]
> Direct use of programing language specific building and testing commands (e.g. `go build`, `go test`, `npm run build`) is prohibited.
> Always use the scripts in `scripts/process/`.

### prompts/
Source of truth for coding agent configuration and project documentation.
- `memory/: Project Memory (design knowledge base)
- `manifest/`: Common IR definitions for multi-tool agent configuration
- `rules/`: Coding standards, testing rules, planning rules
- `phases/`: Phased development specifications and plans
