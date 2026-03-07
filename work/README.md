# work

This directory is reserved for **temporary working directories and development worktrees**.

It is primarily used during development when working with:

* Git worktrees
* Parallel feature development
* Automated agent sessions
* Temporary experimentation

## Typical Usage

Developers or automation tools may create Git worktrees here:

```
git worktree add work/feature-auth feature/auth
git worktree add work/feature-report feature/report
```

Each worktree provides an independent working copy of the repository,
allowing multiple features or tasks to be developed in parallel.

## Example Structure

```
work/
  feature-auth/
  feature-report/
  experiment-x/
```

Each subdirectory behaves as an independent working directory.

## Important Notes

* The contents of this directory are **temporary**.
* Worktrees can be safely removed when development tasks are completed.
* This directory is typically **excluded from version control**.

## Recommended Workflow

1. Create a new worktree inside `work/`.
2. Launch a dev container within the worktree.
3. Run development tools or agents (such as code generation assistants).
4. Merge the results back into the main repository.

This approach enables **parallel development and isolated experimentation**.
