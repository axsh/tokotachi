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

- **Language**: Go (v1.21+)- **Architecture**: Modular / Layered Architecture
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
Vertical slices of the project. Each directory contains a feature module with
its own `go.mod` and application-specific logic.

### shared/libs/
Reusable library packages shared across feature modules. Follows the
"Accept Interfaces, Return Structs" principle. Dependencies flow inward
(features depend on shared, not vice versa).

### scripts/
Build and utility scripts that automate development workflows.
- `process/`: Pipeline scripts (build, test, integration test)
- `utils/`: Helper scripts (status display, validation)

> [!IMPORTANT]
> Direct use of `go build`, `go test`, `npm run build` is prohibited.
> Always use the scripts in `scripts/process/`.

### prompts/
Source of truth for coding agent configuration and project documentation.
- `memory/: Project Memory (design knowledge base)
- `manifest/`: Common IR definitions for multi-tool agent configuration
- `rules/`: Coding standards, testing rules, planning rules
- `phases/`: Phased development specifications and plans

### .agent/
Antigravity-specific configuration. Contains workflows and rules that are
consumed by the Antigravity coding agent. These are migration targets
that will be managed via `prompts/manifest/` in the future.

## Dependency Direction

```
features/myprog  -->  shared/libs
       |
       v
  prompts/rules (referenced by agents)
  .agent/ (consumed by Antigravity)
```

The dependency direction is strictly inward: feature modules may depend
on shared libraries, but shared libraries must not depend on features.
