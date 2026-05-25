// Package git provides thin wrappers around the host's git binary.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Cmd executes a git command in dir and returns trimmed stdout.
// Any error is wrapped with the stderr output for clear diagnostics.
func Cmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n  stderr: %s",
			strings.Join(args, " "),
			err,
			strings.TrimSpace(stderr.String()),
		)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsDirty reports whether the worktree at dir has uncommitted changes.
func IsDirty(dir string) (bool, error) {
	out, err := Cmd(dir, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("checking dirty state: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// IsMerging reports whether a merge is in progress in the worktree at dir.
// It checks for the presence of MERGE_HEAD via git rev-parse.
func IsMerging(dir string) (bool, error) {
	_, err := Cmd(dir, "rev-parse", "--verify", "MERGE_HEAD")
	return err == nil, nil
}

// TagExists reports whether the given tag exists in the repository.
func TagExists(projectRoot, tag string) (bool, error) {
	out, err := Cmd(projectRoot, "tag", "-l", tag)
	if err != nil {
		return false, fmt.Errorf("checking tag: %w", err)
	}
	return strings.TrimSpace(out) == tag, nil
}

// IsMerged reports whether branch is a full ancestor of intoBranch.
func IsMerged(projectRoot, branch, intoBranch string) (bool, error) {
	_, err := Cmd(projectRoot, "merge-base", "--is-ancestor", branch, intoBranch)
	if err != nil {
		// Non-zero exit means not an ancestor — not an error condition.
		return false, nil
	}
	return true, nil
}
