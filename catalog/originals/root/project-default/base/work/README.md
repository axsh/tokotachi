# work

This directory is reserved for temporary working directories and development worktrees.

It is primarily used during development when working with:

* Git worktrees
* Parallel feature development
* Automated agent sessions
* Temporary experimentation

## Typical Usage

```
devctl up <branch> <feature>
```

This creates a worktree under `work/<branch>/` for isolated development.

## Important Notes

* The contents of this directory are **temporary**.
* Worktrees can be safely removed when development tasks are completed.
* This directory is **excluded from version control** via `.gitignore`.
