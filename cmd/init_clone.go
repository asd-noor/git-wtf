package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var initCloneCmd = &cobra.Command{
	Use:   "clone <url> [project-dir]",
	Short: "Clone a remote repository and initialize a git-wtf project",
	Long: `Bare-clones the remote URL into .bare, restores remote tracking refs,
and adds master/develop worktrees.

The project directory is derived from the URL basename if not supplied.
If master or develop branches cannot be inferred, you will be prompted
to select them interactively.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runInitClone,
}

func init() {
	initCmd.AddCommand(initCloneCmd)
}

func runInitClone(_ *cobra.Command, args []string) error {
	url := args[0]
	projectName := urlToDir(url)
	if len(args) == 2 {
		projectName = args[1]
	}
	projectDir, err := filepath.Abs(projectName)
	if err != nil {
		return fmt.Errorf("resolving project directory: %w", err)
	}

	// 1. Create project directory.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	// 2. Bare-clone into .bare.
	fmt.Printf("Cloning %s...\n", url)
	if _, err := git.Cmd(projectDir, "clone", "--bare", url, ".bare"); err != nil {
		return fmt.Errorf("cloning repository: %w", err)
	}

	// 3. Write .git redirect file.
	if err := writeGitRedirect(projectDir); err != nil {
		return err
	}

	// 4. Restore remote tracking refs (bare clones use heads/*, not remotes/*).
	if _, err := git.Cmd(projectDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"); err != nil {
		return fmt.Errorf("configuring remote tracking refs: %w", err)
	}
	if _, err := git.Cmd(projectDir, "fetch"); err != nil {
		return fmt.Errorf("fetching remote refs: %w", err)
	}

	// 5. Interactive branch resolution (include remote branches in the list).
	bs, err := project.ResolveBranches(projectDir, true)
	if err != nil {
		return err
	}

	// 6. Add worktrees (creates local branch tracking origin/<branch> if needed).
	if err := addWorktree(projectDir, "master", bs.Master); err != nil {
		return err
	}
	if err := addWorktree(projectDir, "develop", bs.Develop); err != nil {
		return err
	}

	fmt.Printf("✓ Cloned and initialized git-wtf project at %s\n", projectDir)
	fmt.Printf("  master → %s  |  develop → %s\n", bs.Master, bs.Develop)
	return nil
}

// urlToDir derives a local directory name from a git remote URL.
func urlToDir(url string) string {
	base := filepath.Base(url)
	return strings.TrimSuffix(base, ".git")
}
