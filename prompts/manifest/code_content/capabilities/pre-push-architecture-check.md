---
apiVersion: agent.meta/v1
kind: capability
id: pre-push-architecture-check
title: Pre-Push Architecture Check
description: >-
  Before git push, inspect all committed changes and determine whether they
  contain architecture-relevant knowledge that should be recorded via
  record-architecture-knowledge skill.
paths:
  - "internal/**"
  - "pkg/**"
  - "cmd/**"
  - "scripts/**"
  - ".agents/**"
  - "prompts/manifest/**"
references:
  - "prompts/memory/index.md"
  - "prompts/manifest/code_content/capabilities/record-architecture-knowledge.md"
scripts:
  - "scripts/code/agent/notify.sh"
body: inline
---

# Pre-Push Architecture Check

You are an AI coding agent working in this repository.

Before running `git push`, you **MUST** inspect the changes being pushed and decide
whether they contain architecture-relevant knowledge that should be recorded
for future maintainers.

Your goal is not to summarize the implementation.
Your goal is to capture architectural knowledge that will help a future developer understand:

- Why the system has this structure
- Where responsibilities and boundaries are
- What constraints must not be broken
- What design decisions were made
- What trade-offs or temporary decisions exist
- What future changes must be careful about

## Step 1: Inspect the Changes Being Pushed

Run the following commands to understand what will be pushed:

```bash
git log --oneline origin/HEAD..HEAD
git diff origin/HEAD..HEAD --stat
git diff origin/HEAD..HEAD
```

If there are no unpushed commits, report the following and stop:

```text
Architecture intake: no unpushed commits found. Skipping.
```

## Step 2: Determine Whether an Architecture Note Is Needed

Create an architecture note **only when** the changes being pushed include
at least one of the following architecture signals:

1. A new module, package, directory, service, command, API, component,
   database table, queue, worker, agent, or subsystem was added.
2. Responsibilities were moved, split, merged, or clarified.
3. A boundary was introduced or changed.
   Examples:
   - API boundary
   - Package boundary
   - Process boundary
   - Storage boundary
   - UI/backend boundary
   - Human/AI boundary
   - External service boundary
4. Dependency direction changed or should be protected.
5. State ownership changed.
   Examples:
   - Memory to database
   - Database to cache
   - Synchronous state to asynchronous job state
   - Local state to shared state
6. Data ownership, lifecycle, source of truth, or consistency rule changed.
7. A non-obvious design decision was made.
8. A rejected alternative is visible or can be inferred from the implementation.
9. A temporary implementation, shortcut, or technical debt was introduced.
10. An invariant or rule must remain true for the system to work.
11. Error handling, retry, timeout, transaction, idempotency, locking,
    or recovery behavior changed.
12. Authentication, authorization, permission, policy,
    or trust boundary changed.
13. The change affects future extensibility, plugin structure,
    provider abstraction, agent behavior, or automation flow.
14. The change would be hard to understand later by reading only the code.

If **none** of these apply, report the following and stop:

```text
Architecture intake: no architecture-relevant changes detected.
Reason: <brief reason>
```

## Step 3: What NOT to Record

Do **not** create architecture notes for:

- Trivial refactoring
- Formatting-only changes
- Local implementation details obvious from the code
- Bug fixes that do not change architectural assumptions
- Comments that merely repeat the code
- Temporary debug code
- Mechanical renames with no structural meaning

Architecture notes must not become noisy commit summaries.

## Step 4: Record via record-architecture-knowledge

If architecture signals were detected in Step 2,
use the **record-architecture-knowledge** skill to record the knowledge.

Read the `record-architecture-knowledge` capability for full details on:

- Required command invocation and flags
- Category flag selection
- How to write good architectural notes
- Dry-run for uncertain cases
- Report format

## Step 5: Final Check Before Push

Before allowing `git push` to proceed, confirm:

1. All unpushed commits were inspected.
2. Architecture-relevant signals were evaluated against the 14-item checklist.
3. record-architecture-knowledge was invoked if architecture signals were detected.
4. Report was produced (either "recorded" or "no update").

## Interaction Rules

- Do not ask the user whether to create a note unless the change is ambiguous
  and creating the note would be misleading.
- Prefer recording with a `PROVISIONAL` prefix in the note when the architectural
  signal is important but partially inferred.
- Do not modify production code as part of this architecture intake.
