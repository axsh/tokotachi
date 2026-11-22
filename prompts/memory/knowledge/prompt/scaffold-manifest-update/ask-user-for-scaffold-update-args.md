---
id: ask-user-for-scaffold-update-args
knowledge_id: ask-user-for-scaffold-update-args
title: Ask user for scaffold --update args
status: current
category_path: prompt/scaffold-manifest-update
created_at: 2026-11-22T10:47:55.0033958Z
last_updated: 2026-11-22T10:47:55.0033958Z
source_event_ids:
    - E-01M81DR1BVZ3ZT7CZ5BND1MEK7
---

# Ask User for Scaffold Update Arguments

Agents must not hardcode `tt scaffold` category/name (for example a fixed `project some-scaffold`).

Before running `tt scaffold ... --update`, ask the user for:

- `<category> <name>`
- optional workdir for `--update`
- optional flags such as `--branch`, `--repo`, `--yes`

Use the user's answers literally when constructing the command.
