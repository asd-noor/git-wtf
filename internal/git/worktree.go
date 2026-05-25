package git

import (
	"fmt"
	"strings"
)

// Worktree represents a single entry from `git worktree list --porcelain`.
type Worktree struct {
	Path   string // absolute path to the worktree directory
	Commit string // current HEAD commit hash
	Branch string // full ref, e.g. refs/heads/develop (empty for detached HEAD)
}

// ListWorktrees returns all active worktrees for the project.
func ListWorktrees(projectRoot string) ([]Worktree, error) {
	out, err := Cmd(projectRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}
	return parseWorktrees(out), nil
}

// parseWorktrees parses the porcelain output of `git worktree list`.
// Records are separated by blank lines; each record has key-prefixed lines.
func parseWorktrees(raw string) []Worktree {
	var result []Worktree
	for block := range strings.SplitSeq(raw, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var wt Worktree
		for line := range strings.SplitSeq(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "HEAD "):
				wt.Commit = strings.TrimPrefix(line, "HEAD ")
			case strings.HasPrefix(line, "branch "):
				wt.Branch = strings.TrimPrefix(line, "branch ")
			}
		}
		if wt.Path != "" {
			result = append(result, wt)
		}
	}
	return result
}
