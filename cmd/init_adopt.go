package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var initAdoptCmd = &cobra.Command{
	Use:   "adopt [dir]",
	Short: "Convert an existing local git clone into a git-wtf project",
	Long: `Renames .git to .bare, writes a .git redirect file, restores remote
tracking refs, and adds master/develop worktrees.

Defaults to the current directory if no path is given.
If master or develop branches cannot be inferred, you will be prompted
to select them interactively.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitAdopt,
}

func init() {
	initCmd.AddCommand(initAdoptCmd)
}

func runInitAdopt(_ *cobra.Command, args []string) error {
	dir := "."
	if len(args) == 1 {
		dir = args[0]
	}
	projectDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	// 1. Verify this is a regular clone (has a .git directory, not a file).
	gitPath := filepath.Join(projectDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%s is not a regular git clone (.git directory not found)", projectDir)
	}

	// Guard: don't adopt an already-adopted project.
	if _, err := os.Stat(filepath.Join(projectDir, ".bare")); err == nil {
		return fmt.Errorf("already a git-wtf project (.bare already exists)")
	}

	// 2. Resolve branches while .git is still intact (before any filesystem changes).
	// If the user cancels or branches cannot be inferred, we exit cleanly here.
	bs, err := project.ResolveBranches(projectDir, false)
	if err != nil {
		return err
	}

	// 3. Rename .git → .bare (point of no return — all validation has passed).
	barePath := filepath.Join(projectDir, ".bare")
	if err := os.Rename(gitPath, barePath); err != nil {
		return fmt.Errorf("renaming .git to .bare: %w", err)
	}

	// 4. Write .git redirect file.
	if err := writeGitRedirect(projectDir); err != nil {
		return err
	}

	// 5. Mark the repository as bare.
	if _, err := git.Cmd(projectDir, "config", "--file", ".bare/config", "core.bare", "true"); err != nil {
		return fmt.Errorf("setting core.bare: %w", err)
	}

	// 6. Restore remote tracking refs (best-effort — no remote is not fatal).
	if _, err := git.Cmd(projectDir, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"); err != nil {
		fmt.Println("  ⚠ No remote 'origin' found, skipping remote tracking setup.")
	} else if _, err := git.Cmd(projectDir, "fetch"); err != nil {
		fmt.Println("  ⚠ git fetch failed — you may need to fetch manually.")
	}

	// 7. Add worktrees.
	if err := addWorktree(projectDir, "master", bs.Master); err != nil {
		return err
	}
	if err := addWorktree(projectDir, "develop", bs.Develop); err != nil {
		return err
	}

	fmt.Printf("✓ Adopted git-wtf project at %s\n", projectDir)
	fmt.Printf("  master → %s  |  develop → %s\n", bs.Master, bs.Develop)
	return nil
}
