package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var initAdoptCmd = &cobra.Command{
	Use:   "adopt [dir]",
	Short: "Convert an existing local git clone into a git-wtf project",
	Long: `Checks out the master branch at the project root, adds a develop
worktree under .wtf/, and excludes .wtf/ from git tracking via
.git/info/exclude.

If no directory is given you will be prompted (defaults to current directory).
If master or develop branches cannot be inferred, you will be prompted
to select them interactively.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitAdopt,
}

func init() {
	initCmd.AddCommand(initAdoptCmd)
}

func runInitAdopt(_ *cobra.Command, args []string) error {
	var dirInput string

	if len(args) == 1 {
		dirInput = args[0]
	} else {
		dirInput = "."
		if err := huh.NewInput().
			Title("Directory to adopt").
			Value(&dirInput).
			Run(); err != nil {
			return fmt.Errorf("reading directory: %w", err)
		}
	}

	projectDir, err := filepath.Abs(dirInput)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	// 1. Verify this is a regular git clone (has a .git directory, not a file).
	gitPath := filepath.Join(projectDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%s is not a regular git clone (.git directory not found)", projectDir)
	}

	// 2. Guard: don't adopt an already-adopted project.
	if _, err := os.Stat(filepath.Join(projectDir, ".wtf")); err == nil {
		return fmt.Errorf("already a git-wtf project (.wtf already exists)")
	}

	// 3. Guard: working tree must be clean before checkout.
	if dirty, err := git.IsDirty(projectDir); err != nil {
		return err
	} else if dirty {
		return fmt.Errorf("working tree has uncommitted changes — commit or stash them first")
	}

	// 4. Resolve branches before making any changes.
	// If the user cancels or branches cannot be inferred, we exit cleanly here.
	bs, err := project.ResolveBranches(projectDir, false)
	if err != nil {
		return err
	}

	// 5. Checkout the master branch — the project root serves as the master
	// working tree so it must be on the correct branch.
	if _, err := git.Cmd(projectDir, "checkout", bs.Master); err != nil {
		return fmt.Errorf("checking out %s: %w", bs.Master, err)
	}

	// 6. Add develop worktree under .wtf/.
	if err := addWorktree(projectDir, ".wtf/develop", bs.Develop); err != nil {
		return err
	}

	// 7. Exclude .wtf/ locally without modifying .gitignore.
	if err := addToExclude(projectDir, ".wtf"); err != nil {
		return err
	}

	// Record branch names in git config for use by all subsequent commands.
	if err := project.WriteBranches(projectDir, bs.Master, bs.Develop); err != nil {
		return err
	}

	fmt.Printf("\u2713 Adopted git-wtf project at %s\n", projectDir)
	fmt.Printf("  master \u2192 %s (root)  |  develop \u2192 %s (.wtf/develop/)\n", bs.Master, bs.Develop)
	return nil
}
