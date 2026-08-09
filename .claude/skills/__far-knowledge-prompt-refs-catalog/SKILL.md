---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-prompt-refs-catalog
title: "Far-Knowledge: Prompt Refs Catalog Command"
description: >-
  Cross-cutting knowledge about tt prompt refs: load/parse-only JSON catalog
  of logical {{kind:id}} references without emit, digest, or resolved paths.
user_visible: false
manual_only: false
status: current
body: inline
---

# Prompt Refs Catalog Command

## Command Boundary

`tt prompt refs` is a load/parse-only subcommand. It must call `LoadConfig` + `ParseAllEntities` and must not call emit, digest, or write deploy artifacts.

Existing `tt prompt compile --dry-run` remains a resolved-manifest YAML dump and must not be overloaded as a refs catalog.

## Logical Refs Only

Catalog entries are `{file, kind, id, ref}` where `ref` is the logical string `{{kind:id}}` for `policy` / `procedure` / `capability`.

Resolved editor paths (for example `.claude/rules/…` or `SKILL.md` deploy locations) must not appear in the JSON. Path flags may change which entities are discovered, but must not change the `ref` string for a given kind+id.

## Fail Closed on Parse Errors

If `ParseAllEntities` returns any parse error, `tt prompt refs` must fail the whole command with a non-zero exit. Partial catalogs are not allowed.
