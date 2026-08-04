---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-prompt-manifest-selection-tags
title: "Far-Knowledge: Manifest Selection Tags for Compile/Deploy/Update"
description: >-
  Cross-cutting knowledge about selection tags on code_content manifests,
  --tags/--tag-refs CLI behavior, include/strict reference modes, selected
  markers in resolved manifests, and digest inclusion of tag context.
user_visible: false
manual_only: false
status: current
body: inline
---

# Manifest Selection Tags for Compile/Deploy/Update

## Selection Semantics

`code_content` Manifests (policy / procedure / capability / skip) carry optional frontmatter `tags`.

- If `tags` is omitted, the entity implicitly has the `baseline` tag.
- If `tags` is present, that set **replaces** the implicit baseline (it does not merge). To keep baseline selection, include `baseline` explicitly (example: `tags: [baseline, test]`).
- Empty `tags: []` is rejected by schema/validation.
- Tag names must be kebab-case.

Non-taggable kinds (`guard` / `worker` / `bundle` / `target`) and all `memory_docs` are never filtered by tags; they are always selected.

## CLI and Environment

`prompt compile`, `prompt deploy`, and `prompt update` share:

- `--tags <a,b,...>` (OR match) with priority: CLI > `TT_TAGS` > implicit `baseline`
- `--tag-refs include|strict` with priority: CLI > `TT_TAG_REFS` > `include`

Comma-separated tag values are trimmed; duplicates are dropped with a WARNING.

## Reference Modes

- `include` (default): after tag matching, follow `uses_capabilities` and mark referenced capabilities selected even when their tags do not match.
- `strict`: if a tag-matched entity references a capability that does not match RequestedTags, compilation fails.

## Resolved Output and Emit

Resolved manifests keep **all** entities and mark each with `selected: true|false`. Emitters emit only selected policy / capability / procedure entities.

## Digest

Deploy digest hashing includes normalized, sorted RequestedTags and the tag-refs mode in addition to all source file contents. Changing tags or tag-refs must not be treated as an unchanged deploy.
