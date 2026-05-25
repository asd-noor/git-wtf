package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var pruneDryRun bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove worktrees whose remote branch has been deleted",
	Long: `Runs git fetch --prune origin, then removes every ephemeral worktree
under .wtf/work/, .wtf/release/, or .wtf/hotfix/ whose corresponding remote
branch no longer exists (typically because the PR was merged).

Dirty worktrees are skipped with a warning — commit or stash changes first.

Requires an 'origin' remote. Use --dry-run to preview without making changes.`,
	Args: cobra.NoArgs,
	RunE: runPrune,
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "show what would be removed without making any changes")
	rootCmd.AddCommand(pruneCmd)
}

// pruneCandidate holds a worktree identified for potential removal.
type pruneCandidate struct {
	path         string // absolute path to the worktree directory
	worktreePath string // relative path from root, e.g. .wtf/work/my-feature
	branchName   string // branch name, e.g. work/my-feature
	dirty        bool
	unmerged     bool // not yet in develop or master history
}

func runPrune(_ *cobra.Command, _ []string) error {
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	// 1. Require an origin remote — prune is meaningless without one.
	if _, err := git.Cmd(root, "remote", "get-url", "origin"); err != nil {
		return fmt.Errorf("no 'origin' remote configured — prune checks remote branch state and requires one")
	}

	// 2. Fetch and prune stale remote tracking refs.
	fmt.Println("Fetching remote refs...")
	if _, err := git.Cmd(root, "fetch", "--prune", "origin"); err != nil {
		return fmt.Errorf("git fetch --prune origin: %w", err)
	}

	// 3. List all active worktrees.
	worktrees, err := git.ListWorktrees(root)
	if err != nil {
		return err
	}

	// 4. Identify candidates: ephemeral worktrees under .wtf/<namespace>/<name>
	// whose remote branch no longer exists.
	var candidates []pruneCandidate
	for _, wt := range worktrees {
		rel, err := filepath.Rel(root, wt.Path)
		if err != nil {
			continue
		}

		// Expect exactly three slash-separated parts: .wtf / namespace / name.
		parts := strings.SplitN(filepath.ToSlash(rel), "/", 3)
		if len(parts) != 3 || parts[0] != ".wtf" {
			continue
		}
		namespace := parts[1]
		if namespace != "work" && namespace != "release" && namespace != "hotfix" {
			continue
		}

		// Skip detached HEAD worktrees — no branch to check.
		if wt.Branch == "" {
			continue
		}

		branchName := strings.TrimPrefix(wt.Branch, "refs/heads/")

		// Remote tracking ref still exists — branch is live, skip.
		if _, err := git.Cmd(root, "rev-parse", "--verify", "refs/remotes/origin/"+branchName); err == nil {
			continue
		}

		dirty, _ := git.IsDirty(wt.Path)
		mergedIntoDevelop, _ := git.IsMerged(root, branchName, "develop")
		mergedIntoMaster, _ := git.IsMerged(root, branchName, "master")

		candidates = append(candidates, pruneCandidate{
			path:         wt.Path,
			worktreePath: rel,
			branchName:   branchName,
			dirty:        dirty,
			unmerged:     !mergedIntoDevelop && !mergedIntoMaster,
		})
	}

	if len(candidates) == 0 {
		fmt.Println("Nothing to prune.")
		return nil
	}

	// 5. Report and act on candidates.
	developDir := filepath.Join(root, ".wtf", "develop")
	skipped := 0
	pruned := 0

	for _, c := range candidates {
		if c.dirty {
			fmt.Printf("  \u26a0 skipping %s \u2014 has uncommitted changes (commit or stash first)\n", c.branchName)
			skipped++
			continue
		}
		if c.unmerged {
			fmt.Printf("  \u26a0 %s \u2014 remote branch gone but not merged into develop or master\n", c.branchName)
		}

		if pruneDryRun {
			fmt.Printf("  ~ would remove %s\n", c.branchName)
			pruned++
			continue
		}

		// Remove the worktree.
		if _, err := git.Cmd(root, "worktree", "remove", c.worktreePath); err != nil {
			fmt.Printf("  \u2717 failed to remove worktree %s: %v\n", c.branchName, err)
			skipped++
			continue
		}

		// Delete the branch. Use -D for unmerged branches (remote deletion
		// is intentional; the user was already warned above).
		deleteFlag := "-d"
		if c.unmerged {
			deleteFlag = "-D"
		}
		if _, err := git.Cmd(developDir, "branch", deleteFlag, c.branchName); err != nil {
			fmt.Printf("  \u2717 failed to delete branch %s: %v\n", c.branchName, err)
			skipped++
			continue
		}

		fmt.Printf("  \u2713 removed %s\n", c.branchName)
		pruned++
	}

	// 6. Summary.
	if pruneDryRun {
		fmt.Printf("\nDry run: %d worktree(s) would be removed, %d skipped.\n", pruned, skipped)
	} else {
		fmt.Printf("\nPruned %d worktree(s), %d skipped.\n", pruned, skipped)
	}
	return nil
}
