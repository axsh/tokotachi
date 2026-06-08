---
trigger: always_on
---


Before changing architecture-sensitive code, read prompts/memory/index.md.
After such changes, update the relevant architecture document.
If unsure where to write, append to prompts/memory/inbox.md.

## Mandatory Notify After Architecture-Impacting Changes

When you complete a coherent task boundary (e.g. after `git commit`, before `git push`),
you **MUST** invoke the **record-architecture-knowledge** skill if any of the following occurred:

- A new Go package (`internal/`, `pkg/`, `cmd/`) was added or removed
- A new CLI subcommand or API endpoint was introduced
- Data models or database schemas were added or modified
- Module boundaries or dependency relationships changed
- New wrapper scripts (`scripts/`) were created
- Agent-facing configuration or workflow files were changed
- A design decision was made that future agents should know about

Do **NOT** skip this step. Even if you are unsure whether the change qualifies,
run notify with `--dry-run` first to inspect the payload.

### Execution Timing

Run notify **once per coherent task boundary**. Typical timing:
1. After completing a logical unit of work and running `git commit`
2. Before running `git push`
3. If multiple commits form one logical change, notify once after the last commit

### How to Record

Use the **record-architecture-knowledge** skill for full details on:

- Required command invocation (`./scripts/code/agent/notify.sh`)
- Required and category flags
- How to write good architectural notes
- Dry-run for uncertain cases
- Command examples and report format

## General Rules

Use notify only to store long-term memory candidates.
Do not edit canonical memory documents for intake.
Do not run `./scripts/code/prompt/compile.sh`, `./scripts/code/prompt/deploy.sh`, or `./scripts/code/prompt/update.sh`
unless the user explicitly asks for consolidation or deployment.
