---
id: prompt-refs-catalog-command
knowledge_id: prompt-refs-catalog-command
title: Prompt Refs Catalog Command
status: current
category_path: prompt/refs-catalog
created_at: 2026-08-09T14:56:50.072131Z
last_updated: 2026-08-09T14:56:50.072131Z
source_event_ids:
    - E-01KZKGDEANVH5KEWRZBRWWGHNT
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
