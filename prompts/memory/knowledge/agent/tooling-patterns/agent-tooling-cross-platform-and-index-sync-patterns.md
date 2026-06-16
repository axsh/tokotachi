---
id: agent-tooling-cross-platform-and-index-sync-patterns
knowledge_id: agent-tooling-cross-platform-and-index-sync-patterns
title: Agent Tooling Cross-Platform and Index Sync Patterns
status: current
category_path: agent/tooling-patterns
created_at: 2026-06-15T15:35:16.3426567Z
last_updated: 2026-06-15T15:35:16.3426567Z
source_event_ids:
    - E-01KV5VKD38CR7RKEMXDQCG9CAQ
---

# Agent Tooling Cross-Platform and Index Sync Patterns

## Recursive Directory Walking with Cross-Platform Path Normalization

When walking directory trees that may contain nested structures, use `filepath.WalkDir` instead of `os.ReadDir` (which only reads one level). On Windows, `filepath.Rel` returns backslash-separated paths, so always normalize with `filepath.ToSlash` when the path will be used as a logical identifier (e.g., category path, knowledge ID).

```go
err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
    if err != nil || !d.IsDir() { return nil }
    relPath, _ := filepath.Rel(rootDir, path)
    relPath = filepath.ToSlash(relPath) // Normalize for cross-platform
    // ...
})
```

## File-Move and Index-DB Synchronization

When a file-based state change (e.g., moving from `pending/` to `processed/`) is paired with a SQLite index, both must be updated atomically or at least sequentially:

1. Move the file first (critical operation)
2. Update the DB status immediately after
3. If DB update fails, warn but don't rollback the file move

The `Index.UpdateStatus(eventID, newStatus)` method already existed but was not called from the CLI command. This was the root cause of stale list results.

## Shell Tool Resolution on Windows

For shell scripts that resolve local binaries, always check both `bin/<name>` and `bin/<name>.exe` as Windows requires the `.exe` extension. The fallback order should be:
`$TT_TOOL` env var > PATH > `bin/<name>` > `bin/<name>.exe`
