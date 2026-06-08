---
apiVersion: agent.meta/v1
kind: capability
id: record-architecture-knowledge
title: Record Architecture Knowledge via notify.sh
description: >-
  Shared skill for recording architecture-relevant knowledge into the project
  memory system using ./scripts/code/agent/notify.sh.
  Referenced by architecture-memory policy and pre-push-architecture-check capability.
references:
  - "prompts/memory/index.md"
  - "prompts/memory/schemas/agent-notify-payload.schema.json"
scripts:
  - "scripts/code/agent/notify.sh"
manual_only: true
body: inline
---

# Record Architecture Knowledge via notify.sh

This skill describes how to record architecture-relevant knowledge
into the project's memory system using `./scripts/code/agent/notify.sh`.

This is a shared reference. It is invoked by:

- **architecture-memory** policy: when architecture-impacting changes are detected
  during or after implementation
- **pre-push-architecture-check** capability: when inspecting changes
  before `git push`

## Recording Tool

This project uses `./scripts/code/agent/notify.sh` to record
architecture knowledge into the project's memory system.

You **MUST** use notify.sh. Do **NOT** manually create or edit files
under `prompts/memory/`.

## Required Invocation

```bash
./scripts/code/agent/notify.sh \
  --agent "<your-agent-name>" \
  --summary "<one-line description of the architectural change>" \
  --changed-paths-from-git \
  <category-flags> \
  --note "<note 1>" \
  --note "<note 2>" \
  ...
```

### Required Flags

Every invocation **MUST** include:

- `--agent "<your-agent-name>"` : Your agent name (one of: `antigravity`, `claude-code`, `codex`, `cursor`)
- `--summary "..."` : A one-line description of what changed
- `--changed-paths-from-git` : Automatically detects changed files from git
- At least one category flag (see below)

### Category Flags (pick all that apply)

| Flag | When to use |
|------|-------------|
| `--architecture-impact` | New/removed packages, changed module boundaries, new CLI commands, new API endpoints |
| `--memory-related` | Changes to memory system itself (`prompts/memory/`, `scripts/code/agent/`) |
| `--prompt-related` | Changes to prompt templates (`prompts/manifest/`) |
| `--agent-behavior-related` | Changes to agent rules, workflows, or skills (`.agents/`) |

### Optional Flags

| Flag | Purpose |
|------|---------|
| `--changed-path "<path>"` | Manually specify a changed path (repeatable) |
| `--dry-run` | Preview without recording |
| `--print-payload` | Print the JSON payload (useful with `--dry-run`) |

## Writing Good Notes

Each `--note` value MUST capture a **single architectural proposition**.
Focus on **why**, **what boundary**, **what constraint**, or **what decision** --
not on **what code was written**.

### Good Notes

```text
--note "API handlers must not access the database directly; all DB access goes through usecase services"
--note "The in-memory queue is provisional for local dev; it must not be treated as durable storage"
--note "Task status in the database is the source of truth; queue state is only execution metadata"
--note "The CLI layer parses input and invokes usecase services; it must not perform DB access"
```

### Bad Notes

```text
--note "Added a new function to handle tasks"
--note "Updated some files for better architecture"
--note "Pipeline: A -> B -> C -> D"
```

Each note MUST be a single proposition. Do not pack multiple concepts into one note.

## Note Content Guidelines

When writing notes, prioritize these types of knowledge:

| Type | What to capture |
|------|----------------|
| **Component responsibility** | What this part of the system is responsible for, and what it intentionally excludes |
| **Boundary** | Which layer may call which; which component owns which data; where external systems are isolated |
| **Invariant** | Conditions that must remain true (e.g., "Job execution must be idempotent per job_id") |
| **Design decision** | The selected design and why it was chosen over alternatives |
| **Rejected alternative** | What was considered but not chosen, and why |
| **Risk / temporary** | Provisional implementations, shortcuts, or technical debt that should be revisited |
| **Impact** | What future changes must be careful about; affected tests, APIs, data models |
| **Review trigger** | Under what condition this decision should be reconsidered |

## Confidence Assessment

When the architectural signal is clear from the code, add notes directly.

When the architectural intent is partially inferred,
include a note with a `PROVISIONAL` prefix:

```text
--note "PROVISIONAL: Auth middleware appears to trust the gateway token without re-validation; needs human review"
```

## Dry-Run for Uncertain Cases

If you are unsure whether the change qualifies as architecture-relevant,
run a dry-run first:

```bash
./scripts/code/agent/notify.sh \
  --agent "<your-agent-name>" \
  --summary "<description>" \
  --changed-paths-from-git \
  --architecture-impact \
  --note "<note>" \
  --dry-run --print-payload
```

Inspect the output. If the notes are meaningful, re-run without `--dry-run`.
If the notes feel like noise, skip the recording and report accordingly.

## Command Examples

```bash
# After adding a new package with CLI command
./scripts/code/agent/notify.sh \
  --agent "<your-agent-name>" \
  --summary "Add agent assist handler and task CLI commands" \
  --changed-paths-from-git \
  --architecture-impact \
  --note "New package internal/agent/assist handles agent task intake" \
  --note "New CLI subcommand: tt agent notify" \
  --note "Agent task data stored in .tt/agent/tasks/"

# After modifying data models
./scripts/code/agent/notify.sh \
  --agent "<your-agent-name>" \
  --summary "Add knowledge-atom schema and batch processing types" \
  --changed-paths-from-git \
  --architecture-impact \
  --memory-related \
  --note "KnowledgeAtom struct defined in internal/agent/types.go" \
  --note "Batch storage uses BranchPackageID as partition key"

# After changing agent workflow or rules
./scripts/code/agent/notify.sh \
  --agent "<your-agent-name>" \
  --summary "Update architecture-memory policy to enforce notify.sh usage" \
  --changed-paths-from-git \
  --agent-behavior-related \
  --prompt-related \
  --note "architecture-memory.md policy now lists concrete trigger conditions" \
  --note "record-architecture-knowledge capability created as shared notify.sh reference"

# When unsure if a change qualifies, dry-run first
./scripts/code/agent/notify.sh \
  --agent "<your-agent-name>" \
  --summary "Refactor error handling in auth middleware" \
  --changed-paths-from-git \
  --dry-run --print-payload
```

## Report Format

After running notify.sh, report:

```text
Architecture intake: recorded.
Summary: <summary passed to notify.sh>
Notes:
- <note 1>
- <note 2>
Flags: <flags used>
```

If no recording was made, report:

```text
Architecture intake: no update.
Reason: <brief reason>
```

## Constraints

- Do not modify production code as part of architecture intake.
- Do not create or edit files under `prompts/memory/` directly.
- Do not run `./scripts/code/prompt/compile.sh`, `./scripts/code/prompt/deploy.sh`,
  or `./scripts/code/prompt/update.sh` unless the user explicitly asks.
- Use notify only to store long-term memory candidates.
- Do not edit canonical memory documents for intake.
