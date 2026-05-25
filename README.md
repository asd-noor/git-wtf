# git-vine

`git-vine` manages Git worktrees with a strict Git Flow branching model.
It combines Git worktrees with a strict branch lifecycle for:

- work / feature / bugfix branches
- release branches
- hotfix branches
- bare-repo-based project setup
- stale worktree pruning

The project is designed to be mostly stateless: Git remains the source of truth.

> **Tip:** once the binary is in your `PATH`, Git discovers it automatically —
> every command below works as either `git-vine <cmd>` or `git vine <cmd>`.

## Highlights

- Project root is the master working tree — existing files stay untouched
- All other worktrees live under `.git-vine/` (git-ignored)
- Two initialization modes:
  - `init fresh`
  - `init adopt`
- Opinionated `start` / `finish` lifecycle commands
- Conflict recovery with `--continue` and `--abort`
- `prune` command for stale worktrees
- `version` command with build-time ldflags injection
- Git custom command support: install the binary in your `PATH` and use `git vine`

## Requirements

- Git
- Go 1.26.3 or newer for development

## Installation

### With mise

```sh
mise run install
```

### With Go

```sh
go build -o bin/git-vine .
```

Then place the binary somewhere in your `PATH`, for example:

```sh
install -Dm755 bin/git-vine ~/.local/bin/git-vine
```

Once `git-vine` is in your `PATH`, Git can invoke it as a custom command:

```sh
git vine version
```

## Quick start

### Create a new project

```sh
git-vine init fresh ~/src/my-project
```

### Adopt an existing clone

```sh
cd ~/src/my-project
git-vine init adopt
```

## Command reference

### `init`

Initialize a git-vine project. All inputs are prompted interactively with
sensible defaults when not supplied as arguments.

#### `init fresh`

Create a new git repository and initialize it as a git-vine project.
The project directory is prompted if not supplied (defaults to current directory).

```sh
git-vine init fresh            # prompted
git-vine init fresh ~/src/my-project
```

#### `init adopt`

Convert an existing local clone into a git-vine project in place.
The directory is prompted if not supplied (defaults to `.`).
The working tree must be clean before adopting.

```sh
git-vine init adopt            # prompted, defaults to .
git-vine init adopt ~/src/my-project
```

If `master` or `develop` cannot be inferred automatically, you will be prompted to choose them interactively.

### `work`

Work branches are used for feature development and merge into `develop`.

#### Start

```sh
git-vine work start my-feature
```

Creates an ephemeral worktree for `work/my-feature`.

#### Finish

```sh
git-vine work finish my-feature
```

Merges the branch into `develop`, then removes the worktree and branch.

Conflict recovery flags:

- `--continue` — continue after resolving a merge conflict
- `--abort` — abort an in-progress merge and leave the worktree intact

### `release`

Release branches start from `develop`, merge into `master`, get tagged, and then merge back into `develop`.

#### Start

```sh
git-vine release start 1.2.0
```

#### Finish

```sh
git-vine release finish 1.2.0
```

Conflict recovery flags:

- `--continue` — continue after resolving a merge conflict
- `--abort` — abort an in-progress merge and leave the worktree intact

### `hotfix`

Hotfix branches start from `master`, merge into `master`, get tagged, and then merge back into `develop`.

#### Start

```sh
git-vine hotfix start 1.2.1
```

#### Finish

```sh
git-vine hotfix finish 1.2.1
```

Conflict recovery flags:

- `--continue` — continue after resolving a merge conflict
- `--abort` — abort an in-progress merge and leave the worktree intact

### `prune`

Remove stale ephemeral worktrees whose remote branch has been deleted.

```sh
git-vine prune
```

Use `--dry-run` to preview removals:

```sh
git-vine prune --dry-run
```

Behavior:

- runs `git fetch --prune`
- targets only ephemeral worktrees (`work=*`, `release=*`, `hotfix=*`)
- skips dirty worktrees
- warns if the branch is not merged into `develop` or `master`


### `switch`

Print the path to a git-vine worktree. With no argument, shows an interactive
picker. With a branch name, prints the path directly.

```sh
git-vine switch                # interactive picker
git-vine switch develop        # print .git-vine/develop path
git-vine switch work/my-feature
git-vine switch my-feature     # short form: tries work/<name> automatically
```

Intended for shell integration — wrap in a function to `cd`:

```sh
# Bash / Zsh
gws() { local p; p="$(git-vine switch "$@")" && cd "$p"; }

# Fish
function gws; cd (git-vine switch $argv); end
```

### `version`

Print the current version string.

```sh
git-vine version
```

When built with the project build script, the version is injected from git metadata.

## Worktree layout

`git-vine` uses a bare repository anchor and a conventional directory hierarchy:

- `.git/` — git repository
- `.git-vine/` — git-ignored, holds all other worktrees
  - `develop/` — permanent develop worktree
  - `work/<name>/` — ephemeral feature worktrees
  - `release/<tag>/` — ephemeral release worktrees
  - `hotfix/<tag>/` — ephemeral hotfix worktrees

The project root itself is the master working tree. `.git-vine/` is excluded via
`.git/info/exclude` and never appears in `git status`.

## Development

```sh
go build ./...
go vet ./...
```

If you are using mise tasks:

```sh
mise run build
mise run install
```

## Notes

- `git-vine` is intentionally opinionated and does not add a separate `status` or `sync` command.
- Use native Git commands when you need lower-level control.
- Install the binary in your `PATH` to use Git's custom command discovery (`git vine`).

## License

GPLv3. See `LICENSE` for the full text.
