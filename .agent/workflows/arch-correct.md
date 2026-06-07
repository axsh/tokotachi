---
description: Architecture Correction
---


# Architecture Correction

## Goal

Correct inaccurate or outdated statements in memory documents, based on evidence.

## Steps

### 1. Identify Disputed Statement
- Locate the specific sentence or paragraph in a memory document that needs correction.
- Quote the exact text and note the file path.

### 2. Classify Issue
Assign one of the following categories:
- `factual-error`: Statement contradicts the actual codebase
- `stale-info`: Statement was once true but is no longer accurate
- `policy-change`: A deliberate decision to change a previous policy
- `misclassified`: Information is in the wrong document
- `unsupported-assumption`: Statement lacks evidence
- `wrong-invariant`: An invariant that no longer holds

### 3. Read Architecture Index
- Read `prompts/memory/index.md` to understand the current document structure.
- Identify which document(s) are affected.

### 4. Verify Evidence
The correction MUST be backed by at least one of:
- User instruction (explicit direction from the developer)
- Code (actual implementation in the codebase)
- Tests (test cases that demonstrate the correct behavior)
- Migrations (schema or data changes)
- Existing docs (other memory documents that contradict the disputed statement)

If evidence is insufficient, write the issue to `prompts/memory/open-questions.md` instead of making the correction.

### 5. Apply Minimal Correction
- Edit only the specific statement that is incorrect.
- Do not rewrite surrounding context unless necessary for coherence.
- Preserve the existing document structure and formatting.

### 6. Write ADR If Policy Changed
If the correction involves a `policy-change`:
- Create or update an ADR in `prompts/memory/adr/`.
- If superseding a previous ADR, mark the old one as `superseded_by: <new ADR id>`.
- Record the rationale for the policy change.

### 7. Recompile Generated Files
If frontmatter was changed or a new document was added:
- Run `./scripts/prompt/update.sh --force --target "antigravity"` to recompile and deploy the updated configuration.
