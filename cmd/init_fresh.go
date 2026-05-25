package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"git-wtf/internal/git"
)

var initFreshCmd = &cobra.Command{
	Use:   "fresh [project-dir]",
	Short: "Create a new git-wtf project from scratch",
	Long: `Initializes a git repository, creates an initial commit on master,
adds a develop worktree under .wtf/, and excludes .wtf/ from git
tracking via .git/info/exclude.

If no directory is given you will be prompted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitFresh,
}

func init() {
	initCmd.AddCommand(initFreshCmd)
}

func runInitFresh(_ *cobra.Command, args []string) error {
	var dirInput string

	if len(args) == 1 {
		dirInput = args[0]
	} else {
		cwd, _ := os.Getwd()
		dirInput = cwd
		if err := huh.NewInput().
			Title("Project directory").
			Value(&dirInput).
			Run(); err != nil {
			return fmt.Errorf("reading project directory: %w", err)
		}
	}

	if dirInput == "" {
		return fmt.Errorf("project directory cannot be empty")
	}

	projectDir, err := filepath.Abs(dirInput)
	if err != nil {
		return fmt.Errorf("resolving project directory: %w", err)
	}

	// 1. Create project directory.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	// 2. Initialize repository.
	if _, err := git.Cmd(projectDir, "init"); err != nil {
		return fmt.Errorf("initializing repository: %w", err)
	}

	// 3. Set default branch to master.
	// Using symbolic-ref instead of git init -b for compatibility with git < 2.28.
	if _, err := git.Cmd(projectDir, "symbolic-ref", "HEAD", "refs/heads/master"); err != nil {
		return fmt.Errorf("setting default branch: %w", err)
	}

	// 4. Create initial empty commit to establish the master branch.
	// Requires git user.name and user.email to be configured.
	if _, err := git.Cmd(projectDir, "commit", "--allow-empty", "-m", "Initial commit"); err != nil {
		return fmt.Errorf("creating initial commit: %w\n\nhint: configure git user.name and user.email", err)
	}

	// 5. Add develop worktree branching from HEAD.
	if _, err := git.Cmd(projectDir, "worktree", "add", ".wtf/develop", "-b", "develop", "HEAD"); err != nil {
		return fmt.Errorf("creating develop worktree: %w", err)
	}

	// 6. Exclude .wtf/ locally without modifying .gitignore.
	if err := addToExclude(projectDir, ".wtf"); err != nil {
		return err
	}

	fmt.Printf("\u2713 Initialized new git-wtf project at %s\n", projectDir)
	fmt.Printf("  master (root)  |  develop \u2192 .wtf/develop/\n")
	return nil
}
