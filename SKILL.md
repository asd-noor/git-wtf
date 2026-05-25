---
name: git-wtf
description: >
  Manage Git worktrees following a strict Git Flow branching model using the
  git-wtf CLI. Use this skill when the user wants to start, finish, or switch
  between work/feature branches, release branches, or hotfix branches in a
  git-wtf project, or when they want to initialise a new or existing repository
  as a git-wtf project.
license: GPL-3.0
compatibility: Requires git and the git-wtf binary in PATH. Run as `git-wtf` or `git wtf`.
metadata:
  author: Asaduzzaman Noor
  version: "1.0.1"
  repository: https://github.com/asd-noor/git-wtf
---

# git-wtf Skill

`git-wtf` combines Git worktrees with a Git Flow branching model. The project
root is the **master** working tree. All other worktrees live under `.wtf/`.

## Project layout

```
myproject/
  .git/                    # regular git repository
  .wtf/
    develop/               # permanent develop worktree
    work/<name>/           # ephemeral feature worktrees
    release/<tag>/         # ephemeral release worktrees
    hotfix/<tag>/          # ephemeral hotfix worktrees
  src/                     # project files (untouched on adopt)
```

Branch names are persisted in `.git/config`:

```ini
[git-wtf "branch"]
    master  = master   # or main, trunk, etc.
    develop = develop  # or dev, etc.
```

## Initialisation

### New project

```sh
git wtf init fresh [project-dir]   # prompts if omitted
```

Creates `git init`, sets master branch, creates `.wtf/develop`, writes config.

### Existing clone

```sh
cd my-project
git wtf init adopt                 # prompts for dir, defaults to .
```

Checks out master at root, adds `.wtf/develop`, writes config.
The working tree must be clean before adoption.

## Work (feature) branches

```sh
git wtf work start <name>          # creates .wtf/work/<name> from develop
git wtf work finish <name>         # merges into develop, removes worktree
git wtf work finish <name> --continue  # resume after resolving conflict
git wtf work finish <name> --abort     # abort in-progress merge
```

## Release branches

```sh
git wtf release start <tag>        # creates .wtf/release/<tag> from develop
git wtf release finish <tag>       # merges into master, tags, merges into develop
git wtf release finish <tag> --continue
git wtf release finish <tag> --abort
```

## Hotfix branches

```sh
git wtf hotfix start <tag>         # creates .wtf/hotfix/<tag> from master
git wtf hotfix finish <tag>        # merges into master, tags, merges into develop
git wtf hotfix finish <tag> --continue
git wtf hotfix finish <tag> --abort
```

## Switching between worktrees

```sh
git wtf switch                     # interactive picker (renders to stderr)
git wtf switch develop
git wtf switch master
git wtf switch my-feature          # expands to work/my-feature
git wtf switch work/my-feature
git wtf switch 1.0.0               # tries release/1.0.0 then hotfix/1.0.0
```

Shell integration to `cd` on select:

```sh
# Bash / Zsh
gws() { local p; p="$(git-wtf switch "$@")" && cd "$p"; }

# Fish
function gws; cd (git-wtf switch $argv); end
```

## Pruning stale worktrees

```sh
git wtf prune             # removes worktrees whose remote branch was deleted
git wtf prune --dry-run   # preview without making changes
```

Requires an `origin` remote. Skips dirty worktrees.

## Other commands

```sh
git wtf version           # print version string
```

## Conflict recovery

When a merge conflicts, git-wtf exits with a structured message:

```
✗ Merge conflict in develop

  Resolve it manually:
  1. cd /path/to/.wtf/develop
  2. fix conflicts, then: git add . && git merge --continue
  3. run: git-wtf work finish <name> --continue

  Or to abort: git-wtf work finish <name> --abort
```

The `--continue` flag re-checks that the merge actually landed (via
`git merge-base --is-ancestor`) before cleaning up.

## Key constraints

- **Dirty check**: `finish` commands and `init adopt` block on staged or
  modified tracked files. Untracked files are intentionally ignored.
- **Branch names**: hardcoding `master`/`develop` is wrong; always read from
  `.git/config` via `git config --local git-wtf.branch.master` etc.
- **Duplicate guard**: `start` checks both the worktree directory and the
  branch ref. An orphaned branch (worktree removed but branch not deleted)
  is caught with a clear recovery hint.
- **Release/hotfix finish** checks that both master (root) and develop are
  clean before starting any merge operations.
