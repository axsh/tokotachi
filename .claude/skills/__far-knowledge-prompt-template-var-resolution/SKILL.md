---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-prompt-template-var-resolution
title: "Far-Knowledge: Prompt Template Variable Resolution Across Emitters"
description: >-
  Cross-cutting knowledge about ResolveTemplateVars on all emitters,
  {{target:workflows|rules|skills}} directory vars, and portable policy
  source references without hardcoded .agent/workflows paths.
user_visible: false
manual_only: false
status: current
body: inline
---

# Prompt Template Variable Resolution Across Emitters

## Emitter Responsibility

Every prompt emitter (Claude Code, Codex, Cursor, Antigravity) must call `ResolveTemplateVars` on policy, capability, and procedure body text before writing files.

If only Antigravity resolves templates, other targets leave `{{policy:…}}` / `{{procedure:…}}` / `{{target:…}}` placeholders unresolved in deployed agent files.

## Target Directory Variables

- `{{target:workflows}}` resolves to the Workflows directory when the target defines Workflows; otherwise it falls back to Skills (procedure home).
- `{{target:rules}}` and `{{target:skills}}` expose the target's rules and skills directory roots.

## Portable Policy Sources

Policy source markdown must not hardcode editor-specific paths such as `.agent/workflows/…`.

- Prefer `{{procedure:<id>}}` for file references.
- Prefer `{{target:workflows}}` (and rules/skills vars) for directory references.

## Cursor vs Antigravity Naming

- Cursor policy files use the `.mdc` extension; `project-instructions` keeps that base name.
- Only Antigravity renames `project-instructions` to `instructions.md`.
