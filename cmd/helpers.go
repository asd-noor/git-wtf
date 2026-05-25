package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"git-wtf/internal/git"
)

// mustCwd returns the current working directory or panics.
func mustCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("cannot determine working directory: %v", err))
	}
	return cwd
}

// addWorktree creates a worktree at path for branch.
// If the branch does not exist locally, it is created tracking origin/<branch>.
func addWorktree(projectRoot, path, branch string) error {
	_, err := git.Cmd(projectRoot, "rev-parse", "--verify", "refs/heads/"+branch)
	if err == nil {
		_, err = git.Cmd(projectRoot, "worktree", "add", path, branch)
	} else {
		_, err = git.Cmd(projectRoot, "worktree", "add", path, "-b", branch, "origin/"+branch)
	}
	if err != nil {
		return fmt.Errorf("adding worktree %s: %w", path, err)
	}
	return nil
}

// addToExclude appends pattern to .git/info/exclude so the path is ignored
// locally without touching the project's committed .gitignore.
func addToExclude(projectRoot, pattern string) error {
	excludeDir := filepath.Join(projectRoot, ".git", "info")
	if err := os.MkdirAll(excludeDir, 0755); err != nil {
		return fmt.Errorf("creating .git/info: %w", err)
	}
	excludeFile := filepath.Join(excludeDir, "exclude")
	existing, err := os.ReadFile(excludeFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .git/info/exclude: %w", err)
	}
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == pattern {
			return nil // already present
		}
	}
	f, err := os.OpenFile(excludeFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening .git/info/exclude: %w", err)
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s\n", pattern); err != nil {
		return fmt.Errorf("writing to .git/info/exclude: %w", err)
	}
	return nil
}

// cleanupWorktree removes the worktree directory and deletes the branch.
// mergeDir is the worktree from which the branch deletion is verified (typically
// .wtf/develop, the final merge target in all git-wtf flows). Running
// git branch -d from there ensures the safety check is against the correct HEAD.
func cleanupWorktree(root, mergeDir, worktreeName, branchName string) error {
	if _, err := git.Cmd(root, "worktree", "remove", worktreeName); err != nil {
		return fmt.Errorf("removing worktree %s: %w", worktreeName, err)
	}
	if _, err := git.Cmd(mergeDir, "branch", "-d", branchName); err != nil {
		return fmt.Errorf("deleting branch %s: %w", branchName, err)
	}
	fmt.Printf("✓ Finished %s — worktree removed.\n", branchName)
	return nil
}

// abortMerge runs `git merge --abort` in targetDir if a merge is in progress.
func abortMerge(targetDir string) error {
	merging, err := git.IsMerging(targetDir)
	if err != nil {
		return err
	}
	if !merging {
		return fmt.Errorf("no merge in progress in %s", targetDir)
	}
	if _, err := git.Cmd(targetDir, "merge", "--abort"); err != nil {
		return fmt.Errorf("aborting merge: %w", err)
	}
	fmt.Printf("✓ Merge aborted in %s. Working tree preserved.\n", filepath.Base(targetDir))
	return nil
}

// conflictErr returns a structured, actionable error for a merge conflict.
// targetName is the display name (e.g. "develop", "master").
// targetDir is the absolute path to the worktree where the conflict occurred.
func conflictErr(targetName, targetDir, cmdType, name string) error {
	return fmt.Errorf(
		"✗ Merge conflict in %s\n\n"+
			"  Resolve it manually:\n"+
			"  1. cd %s\n"+
			"  2. fix conflicts, then: git add . && git merge --continue\n"+
			"  3. run: git-wtf %s finish %s --continue\n\n"+
			"  Or to abort: git-wtf %s finish %s --abort",
		targetName,
		targetDir,
		cmdType, name,
		cmdType, name,
	)
}

// continueFinishWithTag handles --continue for release and hotfix finish commands.
// It is stage-aware: determines at which step the conflict occurred and resumes
// from the correct point without re-running completed steps.
func continueFinishWithTag(root, tag, cmdType, branchName, worktreeName string, masterDir, developDir string) error {
	// If either directory is still in a merging state, the user must finish resolving first.
	if merging, err := git.IsMerging(masterDir); err != nil {
		return err
	} else if merging {
		return fmt.Errorf(
			"master merge still in progress\n"+
				"  cd %s && git add . && git merge --continue\n"+
				"  then run: git-wtf %s finish %s --continue",
			masterDir, cmdType, tag)
	}
	if merging, err := git.IsMerging(developDir); err != nil {
		return err
	} else if merging {
		return fmt.Errorf(
			"develop merge still in progress\n"+
				"  cd %s && git add . && git merge --continue\n"+
				"  then run: git-wtf %s finish %s --continue",
			developDir, cmdType, tag)
	}

	// Stage detection via tag existence:
	// - No tag → master merged but tag not yet created; proceed from tagging.
	// - Tag exists → check if develop still needs the merge.
	tagExists, err := git.TagExists(root, tag)
	if err != nil {
		return err
	}
	if !tagExists {
		noun := map[string]string{"release": "Release ", "hotfix": "Hotfix "}[cmdType]
		if _, err := git.Cmd(masterDir, "tag", "-a", tag, "-m", noun+tag); err != nil {
			return fmt.Errorf("creating tag %s: %w", tag, err)
		}
	}

	// Merge develop if not already done.
	merged, err := git.IsMerged(root, branchName, "develop")
	if err != nil {
		return err
	}
	if !merged {
		if _, err := git.Cmd(developDir, "merge", "--no-ff", branchName); err != nil {
			return conflictErr("develop", developDir, cmdType, tag)
		}
	}

	return cleanupWorktree(root, developDir, worktreeName, branchName)
}
