---
apiVersion: agent.meta/v1
kind: capability
id: __far-knowledge-agent-tooling-patterns
title: "Far-Knowledge: Agent Tooling Cross-Platform Patterns"
description: >-
  Cross-cutting knowledge about cross-platform file walking,
  index-DB synchronization, and shell tool resolution patterns.
user_visible: false
manual_only: false
status: current
body: inline
---

# Agent Tooling Cross-Platform Patterns

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

The `Index.UpdateStatus(eventID, newStatus)` method in `storage/index.go` handles the DB update.

## Shell Tool Resolution on Windows

For shell scripts that resolve local binaries, always check both `bin/<name>` and `bin/<name>.exe` as Windows requires the `.exe` extension. The fallback order should be:
`$TT_TOOL` env var > PATH > `bin/<name>` > `bin/<name>.exe`
