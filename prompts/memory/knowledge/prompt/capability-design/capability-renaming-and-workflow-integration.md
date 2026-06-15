---
id: capability-renaming-and-workflow-integration
knowledge_id: capability-renaming-and-workflow-integration
title: Capability Renaming and Workflow Integration
status: current
category_path: prompt/capability-design
created_at: 2026-06-15T14:19:28.8147388Z
last_updated: 2026-06-15T14:19:28.8147388Z
source_event_ids:
    - E-01KTJMMW57Y83D9KGBST4TSGNA
---

# Capability Renaming and Workflow Integration

## Capability Renames

The following capabilities were renamed to better communicate their purpose:

- `architecture-memory-intake` -> `pre-push-architecture-check`: Inspects commits before `git push` for architecture signals. Scope changed from pre-commit to pre-push timing.
- `notify-intake` -> `record-architecture-knowledge`: Shared skill for using `record.sh` / `notify.sh` to record knowledge events.

## Workflow Integration

The `execute-implementation-plan` workflow (Section 3.3) now requires `record-architecture-knowledge` to be invoked before `git push`. This ensures architecture-significant changes are captured as intake events.

## Agent Flag Resolution

The `--agent` flag must be passed at runtime (not via template variable) because `.agents/` is shared between multiple agents (antigravity, codex, etc.). Each agent needs to identify itself when recording events.

## TT_TOOL Environment Variable

When the system-wide `tt` binary lacks the `agent` subcommand (older version), the `TT_TOOL` environment variable can be set to point to a local build (e.g., `./bin/tt.exe`) that includes the latest features. The `_resolve_tool.sh` script handles this resolution automatically.
