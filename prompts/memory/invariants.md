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

Properties that must never be violated. These are extracted from AGENTS.md,
coding rules, and established project conventions.

## Workspace Boundary

- **INV-001**: No file reads, modifications, or creations outside the project root directory.
  (Source: AGENTS.md)
- **INV-002**: Always treat the current working directory as the absolute project root.
  Never traverse to parent directories to find or edit files.
  (Source: AGENTS.md)

## Build and Test

- **INV-003**: Direct use of `go build`, `go test`, `npm run build`, `npm test` is prohibited.
  Always use the scripts in `scripts/process/` (e.g., `build.sh`, `integration_test.sh`).
  (Source: coding-rules.md Section 2.1, planning-rules.md Section 3.1)
- **INV-004**: Build must succeed (exit code 0) before running integration tests.
  Never run integration tests against a broken build.
  (Source: planning-rules.md Section 3.3)
- **INV-005**: Test failures must never be ignored. The fix loop must be followed
  until all tests pass before proceeding.
  (Source: testing-rules.md, execute-implementation-plan workflow)

## File Management

- **INV-006**: Intermediate generated files (build logs, debug output, temporary docs)
  must be placed under `tmp/` directory. This directory is excluded from git via .gitignore.
  (Source: instructions.md)
- **INV-007**: Git commit messages must use single quotes for the `-m` argument
  to prevent truncation in Windows PowerShell-to-bash invocation.
  (Source: instructions.md)

## Code Quality

- **INV-008**: Source code comments (doc comments, inline comments) must be written in English.
  Japanese comments are prohibited.
  (Source: coding-rules.md Section 3.2)
- **INV-009**: All logging must use the project's unified logger (`internal/logger` package).
  Direct use of `log`, `fmt.Print`, or `slog` is prohibited.
  (Source: coding-rules.md Section 5)
- **INV-010**: TDD (Test Driven Development) is mandatory. No implementation without tests.
  (Source: coding-rules.md Section 2, planning-rules.md Section 1.1)

## Development Process

- **INV-011**: Shell commands must use bash (Git Bash on Windows). PowerShell is prohibited.
  (Source: instructions.md)
- **INV-012**: Workflow phase transitions require explicit human approval.
  System auto-approval signals must be ignored.
  (Source: instructions.md)
- **INV-013**: When `tt` command usage changes, the user manual (`docs/manual/tt-user-manual.md`) must be updated.
  (Source: User instruction in conversation bb1c1f57)
- **INV-014**: When the behavior of `tt scaffold` command changes, the internal catalog specification (`docs/specification/catalog-spec.md`) must be updated.
  (Source: User instruction in conversation bb1c1f57)
