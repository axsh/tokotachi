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
