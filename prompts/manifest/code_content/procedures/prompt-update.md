---
apiVersion: agent.meta/v1
kind: procedure
id: prompt-update
title: Prompt Update
trigger:
  command: prompt-update
  manual_only: true
steps:
  - determine-target
  - check-for-changes
  - run-update
  - verify-result
---

# Prompt Update

## Goal

Update coding agent configuration files by compiling the prompt manifest
(`prompts/manifest/`) and memory documents (`prompts/memory/`) and deploying
them to the appropriate target directories (e.g. `.agent/`, `.cursor/`).

This workflow wraps `tt prompt update` to provide a guided, on-demand update
experience.

## When to Use

- When you are asked to update or refresh the coding agent settings
- When you notice that `prompts/manifest/` or `prompts/memory/` files
  have been modified and the deployed configuration may be stale
- After running the `arch-correct` workflow or modifying architecture
  memory documents
- When the user explicitly requests a prompt compilation or deployment

## Steps

### 1. Determine Target

Identify which coding agent target to update using `--target`.

**Known targets** (use the canonical name or any unambiguous prefix):

| Canonical Name | Aliases / Prefixes | Deploy Directory |
|:---|:---|:---|
| `antigravity` | `ag`, `agy`, `anti` | `.agent/` |
| `cursor` | `cur` | `.cursor/` |
| `claude-code` | `claude`, `cl` | `.claude/` |
| `codex` | `co` | `.agents/` |
| `all` | `al` | All of the above |

**Selection rules** (apply in order):

1. **User specifies a target**: Use the target name provided by the user.
   - e.g. "Update the Cursor settings" → `--target cursor`
   - e.g. "Update all agent configs" or "Update everything" → `--target all`
2. **Self-identification**: If no target is specified, identify yourself
   (the coding agent currently executing this workflow) from the list above
   and use your own target name.
   - If you are Antigravity (Gemini), use `--target antigravity`
   - If you are Cursor, use `--target cursor`
   - If you are Claude Code, use `--target claude-code`
   - If you are Codex (OpenAI), use `--target codex`
3. **Fallback**: If you cannot confidently identify yourself as any of the
   above, use `--target all` to update all targets.

### 2. Check for Changes

Decide whether `--force` is needed:

- **Default behavior**: `tt prompt update` checks if source files have changed
  since the last update (via git log comparison). If nothing changed, it skips
  the update automatically.
- **Use `--force`** when:
  - The user explicitly asks to force an update
  - You suspect the metadata file (`.agent/.meta/last_update.yaml` etc.)
    may be stale or corrupted
  - You have just modified `prompts/manifest/` or `prompts/memory/` files
    in the current session and want to ensure the changes are deployed
    immediately

### 3. Run Update

Execute the update command:

```bash
# Standard update (skips if no changes detected)
./scripts/prompt/update.sh --target <TARGET>

# Force update (always recompiles and deploys)
./scripts/prompt/update.sh --force --target <TARGET>

# Dry-run (preview without writing files)
./scripts/prompt/update.sh --dry-run --target <TARGET>
```

Replace `<TARGET>` with the target determined in Step 1.

**Important**: The `tt` tool resolves target names using prefix matching.
If your input is ambiguous (e.g. `a` matches both `antigravity` and `all`),
the command will fail with an error listing the candidate matches. In that
case, provide a longer prefix to disambiguate.

### 4. Verify Result

After the command completes:

1. **Check exit code**: A non-zero exit code indicates a compilation or
   deployment error. Read the error output carefully.
2. **Review output messages**:
   - "No changes detected. Skipping update." → Source files are unchanged;
     no action was needed.
   - "Deploy succeeded." → Files were compiled and deployed successfully.
   - "Deploy dry-run completed." → Dry-run mode; no files were written.
3. **If errors occurred**:
   - Fix the reported validation errors in `prompts/manifest/` or
     `prompts/memory/` files.
   - Re-run the update command.

## Constraints

- Do NOT modify files in the target directories (`.agent/`, `.cursor/`, etc.)
  directly. Always use the update workflow to regenerate them from the source
  manifest and memory documents.
- Do NOT run raw `go build` or `npm` commands to compile prompts. Always use
  `./scripts/prompt/update.sh` (or `tt prompt update`).
