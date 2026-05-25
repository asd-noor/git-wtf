package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"git-wtf/internal/git"
	"git-wtf/internal/project"
)

var switchCmd = &cobra.Command{
	Use:   "switch [branch]",
	Short: "Print the path of a git-wtf worktree",
	Long: `Prints the absolute path to the named worktree, or shows an
interactive picker when no argument is given.

Intended for shell integration — wrap in a function to change directory:

  Bash / Zsh:
    gws() { local p; p="$(git-wtf switch "$@")" && cd "$p"; }

  Fish:
    function gws; cd (git-wtf switch $argv); end`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

// switchEntry pairs a display label with a worktree path.
type switchEntry struct {
	label string // e.g. "master", "develop", "work/my-feature"
	path  string // absolute path to the worktree
}

func runSwitch(_ *cobra.Command, args []string) error {
	root, err := project.FindRoot(mustCwd())
	if err != nil {
		return err
	}

	// Read config for the master label; best-effort — fall back if not initialised.
	masterLabel := "master"
	if bs, err := project.ReadBranches(root); err == nil {
		masterLabel = bs.Master
	}

	worktrees, err := git.ListWorktrees(root)
	if err != nil {
		return err
	}

	var entries []switchEntry
	for _, wt := range worktrees {
		rel, err := filepath.Rel(root, wt.Path)
		if err != nil {
			continue
		}
		var label string
		if rel == "." {
			// Root is the master working tree; label with its branch name.
			label = strings.TrimPrefix(wt.Branch, "refs/heads/")
			if label == "" || label == "HEAD" {
				label = masterLabel
			}
		} else {
			// ".wtf/work/my-feature" \u2192 "work/my-feature"
			label = strings.TrimPrefix(filepath.ToSlash(rel), ".wtf/")
		}
		entries = append(entries, switchEntry{label: label, path: wt.Path})
	}

	if len(entries) == 0 {
		return fmt.Errorf("no worktrees found")
	}

	if len(args) == 1 {
		return switchByName(entries, args[0])
	}

	// Interactive picker — render TUI to stderr so only the path reaches stdout.
	opts := make([]huh.Option[string], len(entries))
	for i, e := range entries {
		opts[i] = huh.NewOption(e.label, e.path)
	}
	var selected string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Switch to worktree").
			Options(opts...).
			Value(&selected),
	)).WithOutput(os.Stderr).Run(); err != nil {
		return fmt.Errorf("selecting worktree: %w", err)
	}
	fmt.Println(selected)
	return nil
}

// switchByName resolves a name to a worktree path and prints it.
// Accepts an exact label (e.g. "work/my-feature", "release/1.0.0") or a
// bare name which is tried against all ephemeral namespaces in order.
func switchByName(entries []switchEntry, name string) error {
	for _, e := range entries {
		if e.label == name {
			fmt.Println(e.path)
			return nil
		}
	}
	// Try bare name with each ephemeral namespace prefix.
	for _, prefix := range []string{"work/", "release/", "hotfix/"} {
		for _, e := range entries {
			if e.label == prefix+name {
				fmt.Println(e.path)
				return nil
			}
		}
	}
	return fmt.Errorf("no worktree found for %q", name)
}
