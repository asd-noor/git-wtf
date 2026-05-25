package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"git-wtf/internal/git"
)

var initFreshCmd = &cobra.Command{
	Use:   "fresh <project-dir>",
	Short: "Create a new git-wtf project from scratch",
	Long: `Creates a bare repository and initialises master and develop worktrees.

Requires git >= 2.42 for orphan worktree support.`,
	Args: cobra.ExactArgs(1),
	RunE: runInitFresh,
}

func init() {
	initCmd.AddCommand(initFreshCmd)
}

func runInitFresh(_ *cobra.Command, args []string) error {
	projectDir, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving project directory: %w", err)
	}

	// 1. Create project directory.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	// 2. Initialize bare repository.
	if _, err := git.Cmd(projectDir, "init", "--bare", ".bare"); err != nil {
		return fmt.Errorf("initializing bare repository: %w", err)
	}

	// 3. Write .git redirect so standard git tools work from the project root.
	if err := writeGitRedirect(projectDir); err != nil {
		return err
	}

	// 4. Add master worktree with an orphan branch (no commits exist yet).
	if _, err := git.Cmd(projectDir, "worktree", "add", "--orphan", "-b", "master", "master"); err != nil {
		return fmt.Errorf("creating master worktree: %w", err)
	}

	// 5. Create the initial empty commit to establish history.
	masterDir := filepath.Join(projectDir, "master")
	if _, err := git.Cmd(masterDir, "commit", "--allow-empty", "-m", "Initial commit"); err != nil {
		return fmt.Errorf("creating initial commit: %w", err)
	}

	// 6. Add develop worktree branching from master.
	if _, err := git.Cmd(projectDir, "worktree", "add", "develop", "-b", "develop", "master"); err != nil {
		return fmt.Errorf("creating develop worktree: %w", err)
	}

	fmt.Printf("✓ Initialized new git-wtf project at %s\n", projectDir)
	fmt.Printf("  Worktrees ready: master/  develop/\n")
	return nil
}
