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

	worktreeDir := filepath.Join(root, "release", tag)
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("release '%s' already exists at %s", tag, worktreeDir)
	}

	if _, err := git.Cmd(root, "worktree", "add", "release/"+tag, "-b", "release/"+tag, "develop"); err != nil {
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

	releaseDir := filepath.Join(root, "release", tag)
	masterDir := filepath.Join(root, "master")
	developDir := filepath.Join(root, "develop")
	branchName := "release/" + tag
	worktreeName := "release/" + tag

	if releaseAbort {
		if merging, _ := git.IsMerging(masterDir); merging {
			return abortMerge(masterDir)
		}
		if merging, _ := git.IsMerging(developDir); merging {
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

	// Step 3: Merge into master.
	if _, err := git.Cmd(masterDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr(root, "master", "release", tag)
	}

	// Step 4: Tag — only after a successful master merge.
	if _, err := git.Cmd(masterDir, "tag", "-a", tag, "-m", "Release "+tag); err != nil {
		return fmt.Errorf("creating tag %s: %w", tag, err)
	}

	// Step 5: Merge into develop.
	if _, err := git.Cmd(developDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr(root, "develop", "release", tag)
	}

	return cleanupWorktree(root, developDir, worktreeName, branchName)
}
