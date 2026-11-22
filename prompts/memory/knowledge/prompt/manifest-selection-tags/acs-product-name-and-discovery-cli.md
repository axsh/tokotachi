---
id: acs-product-name-and-discovery-cli
knowledge_id: acs-product-name-and-discovery-cli
title: ACS product name and discovery CLI
status: current
category_path: prompt/manifest-selection-tags
created_at: 2026-11-22T10:09:50.8325437Z
last_updated: 2026-11-22T10:09:50.8325437Z
source_event_ids:
    - E-01M80ZVX8CEYNJVD6MC63YFFM5
    - E-01M810HTQG3VENHKX04BAA00TF
    - E-01M816B9S3AHNYX0WCVJ74EQX7
---

# ACS Product Name and Discovery

ACS (Agent Content Selection) is the product name for Manifest Selection Tags.

## Scope (plan B)

Additional ACS scope is documentation, non-baseline catalog samples, and the `tt prompt tags` discovery CLI. The selection engine itself is not reimplemented.

## Implicit tag

- Implicit selection tag remains `baseline`
- A `dev` tag will not be introduced for this purpose

## Discovery

- `tt prompt tags` lists available tags for discovery
- Selection remains `--tags` / `TT_TAGS` with the existing include/strict reference modes
