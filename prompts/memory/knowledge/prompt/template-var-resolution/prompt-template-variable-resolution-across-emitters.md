---
id: prompt-template-variable-resolution-across-emitters
knowledge_id: prompt-template-variable-resolution-across-emitters
title: Prompt Template Variable Resolution Across Emitters
status: current
category_path: prompt/template-var-resolution
created_at: 2026-08-09T14:56:40.5205299Z
last_updated: 2026-08-09T14:56:40.5205299Z
source_event_ids:
    - E-01KZDC3JKYHPJ6QHVR1AB3JXRT
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
