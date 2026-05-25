package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Manage release branches",
}

var releaseStartCmd = &cobra.Command{
	Use:   "start <tag>",
	Short: "Start a release branch from develop",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseStart,
}

var (
	releaseContinue bool
	releaseAbort    bool
)

var releaseFinishCmd = &cobra.Command{
	Use:   "finish <tag>",
	Short: "Merge release into master and develop, tag, and clean up",
	Args:  cobra.ExactArgs(1),
	RunE:  runReleaseFinish,
}

func init() {
	releaseFinishCmd.Flags().BoolVar(&releaseContinue, "continue", false, "continue after resolving a merge conflict")
	releaseFinishCmd.Flags().BoolVar(&releaseAbort, "abort", false, "abort an in-progress merge and leave worktree intact")
	releaseCmd.AddCommand(releaseStartCmd, releaseFinishCmd)
	rootCmd.AddCommand(releaseCmd)
}

func runReleaseStart(_ *cobra.Command, args []string) error {
	tag := args[0]
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	worktreeDir := filepath.Join(root, ".wtf", "release", tag)
	branchName := "release/" + tag

	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("release '%s' already exists at %s", tag, worktreeDir)
	}
	if _, err := git.Cmd(root, "rev-parse", "--verify", "refs/heads/"+branchName); err == nil {
		return fmt.Errorf("branch %s already exists without a worktree — remove it with: git branch -D %s", branchName, branchName)
	}

	bs, err := project.ReadBranches(root)
	if err != nil {
		return err
	}

	if _, err := git.Cmd(root, "worktree", "add", ".wtf/release/"+tag, "-b", branchName, bs.Develop); err != nil {
		return fmt.Errorf("creating release worktree: %w", err)
	}

	fmt.Printf("✓ Started release/%s\n  Worktree: %s\n", tag, worktreeDir)
	return nil
}

func runReleaseFinish(_ *cobra.Command, args []string) error {
	tag := args[0]
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	releaseDir := filepath.Join(root, ".wtf", "release", tag)
	masterDir := root
	developDir := filepath.Join(root, ".wtf", "develop")
	branchName := "release/" + tag
	worktreeName := ".wtf/release/" + tag

	if releaseAbort {
		if merging, err := git.IsMerging(masterDir); err != nil {
			return err
		} else if merging {
			return abortMerge(masterDir)
		}
		if merging, err := git.IsMerging(developDir); err != nil {
			return err
		} else if merging {
			return abortMerge(developDir)
		}
		return fmt.Errorf("no merge in progress")
	}

	if releaseContinue {
		return continueFinishWithTag(root, tag, "release", branchName, worktreeName, masterDir, developDir)
	}

	// Verify worktree exists.
	if _, err := os.Stat(releaseDir); os.IsNotExist(err) {
		return fmt.Errorf("release '%s' not found at %s", tag, releaseDir)
	}

	// Dirty check.
	if dirty, err := git.IsDirty(releaseDir); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("release/%s has uncommitted changes — commit or stash them first", tag)
	}

	// Read config here — only needed for dirty-check error messages.
	bs, err := project.ReadBranches(root)
	if err != nil {
		return err
	}

	// Both permanent worktrees must be clean before any merges begin.
	if dirty, err := git.IsDirty(masterDir); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("%s has uncommitted changes — commit or stash them first", bs.Master)
	}
	if dirty, err := git.IsDirty(developDir); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("%s has uncommitted changes — commit or stash them first", bs.Develop)
	}

	// Step 3: Merge into master.
	if _, err := git.Cmd(masterDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr("master", root, "release", tag)
	}

	// Step 4: Tag — only after a successful master merge.
	if _, err := git.Cmd(masterDir, "tag", "-a", tag, "-m", "Release "+tag); err != nil {
		return fmt.Errorf("creating tag %s: %w", tag, err)
	}

	// Step 5: Merge into develop.
	if _, err := git.Cmd(developDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr("develop", developDir, "release", tag)
	}

	return cleanupWorktree(root, developDir, worktreeName, branchName)
}
