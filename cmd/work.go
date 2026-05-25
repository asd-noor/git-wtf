package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"git-vine/internal/git"
	"git-vine/internal/project"
)

var workCmd = &cobra.Command{
	Use:   "work",
	Short: "Manage work (feature) branches",
}

var workStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a new work branch from develop",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkStart,
}

var (
	workContinue bool
	workAbort    bool
)

var workFinishCmd = &cobra.Command{
	Use:   "finish <name>",
	Short: "Merge a work branch into develop and clean up",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkFinish,
}

func init() {
	workFinishCmd.Flags().BoolVar(&workContinue, "continue", false, "continue after resolving a merge conflict")
	workFinishCmd.Flags().BoolVar(&workAbort, "abort", false, "abort an in-progress merge and leave worktree intact")
	workCmd.AddCommand(workStartCmd, workFinishCmd)
	rootCmd.AddCommand(workCmd)
}

func runWorkStart(_ *cobra.Command, args []string) error {
	name := args[0]
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	worktreeDir := filepath.Join(root, ".git-vine", "work", name)
	branchName := "work/" + name

	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("work branch '%s' already exists at %s", name, worktreeDir)
	}
	// Guard: branch may exist without a worktree (orphaned from partial cleanup).
	if _, err := git.Cmd(root, "rev-parse", "--verify", "refs/heads/"+branchName); err == nil {
		return fmt.Errorf("branch %s already exists without a worktree — remove it with: git branch -D %s", branchName, branchName)
	}

	bs, err := project.ReadBranches(root)
	if err != nil {
		return err
	}

	if _, err := git.Cmd(root, "worktree", "add", ".git-vine/work/"+name, "-b", branchName, bs.Develop); err != nil {
		return fmt.Errorf("creating work worktree: %w", err)
	}

	fmt.Printf("✓ Started work/%s\n  Worktree: %s\n", name, worktreeDir)
	return nil
}

func runWorkFinish(_ *cobra.Command, args []string) error {
	name := args[0]
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	worktreeDir := filepath.Join(root, ".git-vine", "work", name)
	developDir := filepath.Join(root, ".git-vine", "develop")
	branchName := "work/" + name
	worktreeName := ".git-vine/work/" + name

	if workAbort {
		return abortMerge(developDir)
	}

	if workContinue {
		// Verify the user has finished resolving the conflict.
		if merging, err := git.IsMerging(developDir); err != nil {
			return err
		} else if merging {
			return fmt.Errorf(
				"merge in develop is still in progress\n"+
					"  cd %s && git add . && git merge --continue\n"+
					"  then run: git-vine work finish %s --continue",
				developDir, name)
		}
		// Verify the merge actually landed before removing the branch.
		bs, err := project.ReadBranches(root)
		if err != nil {
			return err
		}
		if merged, err := git.IsMerged(root, branchName, bs.Develop); err != nil {
			return err
		} else if !merged {
			return fmt.Errorf("%s has not been merged into develop — complete the merge before using --continue", branchName)
		}
		return cleanupWorktree(root, developDir, worktreeName, branchName)
	}

	// Verify worktree exists.
	if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
		return fmt.Errorf("work branch '%s' not found at %s", name, worktreeDir)
	}
	// Dirty check.
	if dirty, err := git.IsDirty(worktreeDir); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("work/%s has uncommitted changes — commit or stash them first", name)
	}

	// Merge into develop.
	if _, err := git.Cmd(developDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr("develop", developDir, "work", name)
	}

	return cleanupWorktree(root, developDir, worktreeName, branchName)
}
