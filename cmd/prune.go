package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"git-vine/internal/git"
	"git-vine/internal/project"
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

	developDir := filepath.Join(root, ".wtf", "develop")

	// 1. Require an origin remote — prune is meaningless without one.
	if _, err := git.Cmd(root, "remote", "get-url", "origin"); err != nil {
		return fmt.Errorf("no 'origin' remote configured \u2014 prune checks remote branch state and requires one")
	}

	// 2. Fetch and prune stale remote tracking refs.
	fmt.Println("Fetching remote refs...")
	if _, err := git.Cmd(root, "fetch", "--prune", "origin"); err != nil {
		return fmt.Errorf("git fetch --prune origin: %w", err)
	}

	// 3. Read branch names from config.
	bs, err := project.ReadBranches(root)
	if err != nil {
		return err
	}

	// 4. List all active worktrees.
	worktrees, err := git.ListWorktrees(root)
	if err != nil {
		return err
	}

	// Counts declared here so detection-loop errors are included in the summary.
	skipped := 0
	pruned := 0

	// 5. Identify candidates: ephemeral worktrees under .wtf/<namespace>/<name>
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

		if wt.Branch == "" {
			continue // detached HEAD
		}

		branchName := strings.TrimPrefix(wt.Branch, "refs/heads/")

		if _, err := git.Cmd(root, "rev-parse", "--verify", "refs/remotes/origin/"+branchName); err == nil {
			continue // remote still live
		}

		dirty, err := git.IsDirty(wt.Path)
		if err != nil {
			fmt.Printf("  \u26a0 skipping %s \u2014 could not check dirty state: %v\n", branchName, err)
			skipped++
			continue
		}
		mergedIntoDevelop, err := git.IsMerged(root, branchName, bs.Develop)
		if err != nil {
			fmt.Printf("  \u26a0 skipping %s \u2014 could not check merge state: %v\n", branchName, err)
			skipped++
			continue
		}
		mergedIntoMaster, err := git.IsMerged(root, branchName, bs.Master)
		if err != nil {
			fmt.Printf("  \u26a0 skipping %s \u2014 could not check merge state: %v\n", branchName, err)
			skipped++
			continue
		}

		candidates = append(candidates, pruneCandidate{
			path:         wt.Path,
			worktreePath: rel,
			branchName:   branchName,
			dirty:        dirty,
			unmerged:     !mergedIntoDevelop && !mergedIntoMaster,
		})
	}

	if len(candidates) == 0 && skipped == 0 {
		fmt.Println("Nothing to prune.")
		return nil
	}

	// 6. Report and act on candidates.
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

		if _, err := git.Cmd(root, "worktree", "remove", c.worktreePath); err != nil {
			fmt.Printf("  \u2717 failed to remove worktree %s: %v\n", c.branchName, err)
			skipped++
			continue
		}

		// Verify develop worktree exists before deleting the branch from it.
		if _, err := os.Stat(developDir); err != nil {
			fmt.Printf("  \u2717 .wtf/develop not found \u2014 worktree removed but branch %s was not deleted (recover: git branch -D %s)\n", c.branchName, c.branchName)
			skipped++
			continue
		}

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

	// 7. Summary.
	if pruneDryRun {
		fmt.Printf("\nDry run: %d worktree(s) would be removed, %d skipped.\n", pruned, skipped)
	} else {
		fmt.Printf("\nPruned %d worktree(s), %d skipped.\n", pruned, skipped)
	}
	return nil
}
