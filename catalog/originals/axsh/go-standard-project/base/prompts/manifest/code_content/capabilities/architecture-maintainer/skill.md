---
apiVersion: agent.meta/v1
description: Use when implementation, refactoring, or review may affect architecture. Start from prompts/memory/index.md.
id: architecture-maintainer
kind: capability
manual_only: false
paths:
    - features/**/*
    - shared/**/*
    - prompts/memory/**/*
    - scripts/**/*
    - prompts/manifest/**/*
title: Architecture Maintainer
---
# Architecture Maintainer

## Goal

Keep architecture memory accurate and up-to-date as the codebase evolves.

## Entry Point

Read `prompts/memory/index.md` first. This file contains a routing table
that directs you to the appropriate architecture document based on your task.

## Workflow

1. **Before changing architecture-sensitive code**:
   - Read `prompts/memory/index.md`
   - Identify which architecture documents are relevant
   - Check `prompts/memory/invariants.md` for constraints

2. **After making changes**:
   - Update the relevant architecture document
   - If a design decision was made, record it in `prompts/memory/decisions.md`
   - If an invariant was added or changed, update `prompts/memory/invariants.md`
   - If you added, modified, or deleted any memory documents in `prompts/memory/` (including content or frontmatter changes),
     you MUST run `./scripts/prompt/compile.sh` to compile the memory documents (this converts the information in the `prompts/memory/` folder into skills for the coding agent and verifies metadata integrity).
   - If compilation succeeds without any errors, run `./scripts/prompt/deploy.sh --force` to deploy the changes (this reflects the compiled skills to the active directory where the coding agent's skills are stored).

3. **When unsure**:
   - Write to `prompts/memory/inbox.md`
   - Items will be reviewed and promoted to the appropriate document later

## Constraints

- Do not invent architecture. Document what exists or what was decided.
- If unclear, write to `open-questions.md` rather than making assumptions.
- If a design decision changed, update or create an ADR in `adr/`.
- Keep documents focused: one topic per file in `modules/`.
