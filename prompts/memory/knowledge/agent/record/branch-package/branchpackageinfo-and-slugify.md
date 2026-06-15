---
id: branchpackageinfo-and-slugify
knowledge_id: branchpackageinfo-and-slugify
title: BranchPackageInfo and Slugify
status: current
category_path: agent/record/branch-package
created_at: 2026-06-15T13:53:46.3437296Z
last_updated: 2026-06-15T13:53:46.3437296Z
source_event_ids:
    - E-01KTHNQGQXX4S6M0EETHKRPT0S
---

# BranchPackageInfo and Slugify

## BranchPackageInfo

`BranchPackageInfo` is a structured type defined in `features/tt/internal/agent/types.go` that holds branch package identifiers. It is embedded in intake events as `branch_package` field to provide structured branch context.

Key function `DeriveBranchPackage()` in `features/tt/internal/agent/record/branch.go` creates this struct from `GitInfo` and a `GitExecutor`.

## Slugify

`Slugify()` function in `features/tt/internal/agent/record/branch.go` converts branch names to path-safe slugs by replacing path-unsafe characters (slashes, backslashes, colons, etc.) with dashes. It is used to generate the slug component of `BranchPackageInfo.ID`.

Key behaviors:
- Converts path-unsafe characters to dashes
- Truncates long names to keep paths manageable
- Ensures output is safe for use in file system paths
