---
name: git-vine
description: >
  Manage Git worktrees with a Git Flow branching model using the git-vine CLI
  (also invoked as `git vine`). Activate when the user mentions starting,
  finishing, or switching work/feature, release, or hotfix branches, pruning
  stale worktrees, or initialising a repository as a git-vine project.
license: GPL-3.0
compatibility: Requires git and the git-vine binary in PATH.
metadata:
  author: Asaduzzaman Noor
  version: "1.1.0"
---

# git-vine

## Mental model

- **Project root = master working tree** — never a separate directory.
- **All other worktrees** live under `.git-vine/` (locally git-ignored).
- **Branch names** are stored in `.git/config` under `[git-vine "branch"]`.
  Always read them with `git config --local git-vine.branch.master` — never
  hardcode `master` or `develop`.

## Choosing a command

| User intent | Command |
|---|---|
| New project from scratch | `git vine init fresh [dir]` |
| Existing clone → git-vine | `git vine init adopt [dir]` |
| Start feature work | `git vine work start <name>` |
| Finish feature, merge to develop | `git vine work finish <name>` |
| Cut a release from develop | `git vine release start <tag>` |
| Merge release → master + tag + develop | `git vine release finish <tag>` |
| Cut a hotfix from master | `git vine hotfix start <tag>` |
| Merge hotfix → master + tag + develop | `git vine hotfix finish <tag>` |
| Navigate to a worktree | `git vine switch [name]` |
| Remove worktrees with deleted remote | `git vine prune [--dry-run]` |

## Critical invariants

1. **Dirty check** blocks `finish` commands and `init adopt`.
   Only staged/modified tracked files count — untracked files are intentionally
   ignored (git checkout and merge are not blocked by them).

2. **`finish` checks both permanent worktrees are clean** before any merge
   starts (`release`/`hotfix`). A dirty develop will abort a release finish
   even though the conflict is not in the release branch.

3. **`start` guards both directory and branch ref.** An orphaned branch
   (worktree removed but branch not deleted) blocks re-creation with a clear
   recovery hint: `git branch -D <branch>`.

4. **`--continue` verifies the merge actually landed** (`merge-base
   --is-ancestor`) before cleanup. Running `--continue` after a manual
   `git merge --abort` correctly refuses to clean up.

5. **`switch` outputs only the path to stdout.** The TUI renders to stderr so
   `p=$(git vine switch)` captures just the path. Shell integration:
   ```sh
   gws() { local p; p="$(git vine switch "$@")" && cd "$p"; }
   ```

## Conflict recovery

When a merge conflicts, re-run with `--continue` after resolving:

```sh
cd .git-vine/develop        # or .git-vine/release/<tag>, etc.
git add . && git merge --continue
git vine work finish <name> --continue   # or release/hotfix
```

Or abort entirely:
```sh
git vine work finish <name> --abort
```

`release`/`hotfix finish --continue` is stage-aware: it uses tag existence to
determine whether the master merge or develop merge conflicted and resumes from
the correct step.

## init adopt — pre-conditions

- Working tree must be clean (no staged or unstaged changes to tracked files).
- `ResolveBranches` runs before any filesystem changes — cancelling the
  interactive prompt leaves the repo untouched.
- If develop does not exist, the prompt offers to create it from master.

## prune — when it works

Requires an `origin` remote. Fetches `--prune`, then checks
`refs/remotes/origin/<branch>` for each ephemeral worktree. Dirty worktrees
are skipped. Branches not yet merged into master or develop trigger a warning
and are force-deleted (`-D`) because remote deletion is considered intentional.
