---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-prompt-capability-design
title: "Far-Knowledge: Capability Design Patterns"
description: >-
  Cross-cutting knowledge about capability naming conventions,
  workflow integration, agent flag resolution, and TT_TOOL environment variable usage.
user_visible: false
manual_only: false
status: current
body: inline
---

# Capability Design Patterns

## Capability Naming Conventions

Capabilities were renamed to communicate their purpose clearly:

- `architecture-memory-intake` -> `pre-push-architecture-check`: Inspects commits before `git push` for architecture signals. Scope changed from pre-commit to pre-push timing.
- `notify-intake` -> `record-architecture-knowledge`: Shared skill for using `record.sh` / `notify.sh` to record knowledge events.

## Workflow Integration

The `execute-implementation-plan` workflow (Section 3.3) requires `record-architecture-knowledge` to be invoked before `git push`. This ensures architecture-significant changes are captured as intake events.

## Agent Flag Resolution

The `--agent` flag must be passed at runtime (not via template variable) because `.agents/` is shared between multiple agents (antigravity, codex, etc.). Each agent needs to identify itself when recording events.

## TT_TOOL Environment Variable

When the system-wide `tt` binary lacks the `agent` subcommand (older version), the `TT_TOOL` environment variable can be set to point to a local build (e.g., `./bin/tt.exe`) that includes the latest features. The `_resolve_tool.sh` script handles this resolution automatically.
