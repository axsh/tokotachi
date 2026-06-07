---
apiVersion: agent.meta/v1
kind: policy
id: architecture-memory
title: Architecture Memory Policy
scope: project
activation:
    mode: always
applies_when: Applies when changing architecture-sensitive code or module boundaries
---

Before changing architecture-sensitive code, read prompts/memory/index.md.
After such changes, update the relevant architecture document.
If unsure where to write, append to prompts/memory/inbox.md.

When architecture-impacting or agent-memory-relevant knowledge may have been created,
run `./scripts/code/agent/notify.sh` with appropriate flags once per coherent task boundary.

Use notify only to store long-term memory candidates.
Do not edit canonical memory documents for intake.
Do not run `./scripts/code/prompt/compile.sh`, `./scripts/code/prompt/deploy.sh`, or `./scripts/code/prompt/update.sh`
unless the user explicitly asks for consolidation or deployment.
