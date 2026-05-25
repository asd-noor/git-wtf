package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var hotfixCmd = &cobra.Command{
	Use:   "hotfix",
	Short: "Manage hotfix branches",
}

var hotfixStartCmd = &cobra.Command{
	Use:   "start <tag>",
	Short: "Start a hotfix branch from master",
	Args:  cobra.ExactArgs(1),
	RunE:  runHotfixStart,
}

var (
	hotfixContinue bool
	hotfixAbort    bool
)

var hotfixFinishCmd = &cobra.Command{
	Use:   "finish <tag>",
	Short: "Merge hotfix into master and develop, tag, and clean up",
	Args:  cobra.ExactArgs(1),
	RunE:  runHotfixFinish,
}

func init() {
	hotfixFinishCmd.Flags().BoolVar(&hotfixContinue, "continue", false, "continue after resolving a merge conflict")
	hotfixFinishCmd.Flags().BoolVar(&hotfixAbort, "abort", false, "abort an in-progress merge and leave worktree intact")
	hotfixCmd.AddCommand(hotfixStartCmd, hotfixFinishCmd)
	rootCmd.AddCommand(hotfixCmd)
}

func runHotfixStart(_ *cobra.Command, args []string) error {
	tag := args[0]
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	worktreeDir := filepath.Join(root, ".wtf", "hotfix", tag)
	if _, err := os.Stat(worktreeDir); err == nil {
		return fmt.Errorf("hotfix '%s' already exists at %s", tag, worktreeDir)
	}

	if _, err := git.Cmd(root, "worktree", "add", ".wtf/hotfix/"+tag, "-b", "hotfix/"+tag, "master"); err != nil {
		return fmt.Errorf("creating hotfix worktree: %w", err)
	}

	fmt.Printf("✓ Started hotfix/%s\n  Worktree: %s\n", tag, worktreeDir)
	return nil
}

func runHotfixFinish(_ *cobra.Command, args []string) error {
	tag := args[0]
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	hotfixDir := filepath.Join(root, ".wtf", "hotfix", tag)
	masterDir := root
	developDir := filepath.Join(root, ".wtf", "develop")
	branchName := "hotfix/" + tag
	worktreeName := ".wtf/hotfix/" + tag

	if hotfixAbort {
		if merging, _ := git.IsMerging(masterDir); merging {
			return abortMerge(masterDir)
		}
		if merging, _ := git.IsMerging(developDir); merging {
			return abortMerge(developDir)
		}
		return fmt.Errorf("no merge in progress")
	}

	if hotfixContinue {
		return continueFinishWithTag(root, tag, "hotfix", branchName, worktreeName, masterDir, developDir)
	}

	// Verify worktree exists.
	if _, err := os.Stat(hotfixDir); os.IsNotExist(err) {
		return fmt.Errorf("hotfix '%s' not found at %s", tag, hotfixDir)
	}

	// Dirty check.
	if dirty, err := git.IsDirty(hotfixDir); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("hotfix/%s has uncommitted changes — commit or stash them first", tag)
	}

	// Step 3: Merge into master.
	if _, err := git.Cmd(masterDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr("master", root, "hotfix", tag)
	}

	// Step 4: Tag — only after a successful master merge.
	if _, err := git.Cmd(masterDir, "tag", "-a", tag, "-m", "Hotfix "+tag); err != nil {
		return fmt.Errorf("creating tag %s: %w", tag, err)
	}

	// Step 5: Merge into develop.
	if _, err := git.Cmd(developDir, "merge", "--no-ff", branchName); err != nil {
		return conflictErr("develop", developDir, "hotfix", tag)
	}

	return cleanupWorktree(root, developDir, worktreeName, branchName)
}
