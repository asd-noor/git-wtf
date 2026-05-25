package project

import (
	"fmt"
	"slices"
	"strings"

	"git-vine/internal/git"

	"github.com/charmbracelet/huh"
)

// BranchSet holds the resolved master and develop branch names.
type BranchSet struct {
	Master  string
	Develop string
}

// ResolveBranches returns the master and develop branch names.
// Branches that can be auto-detected ("master"/"main" for master,
// "develop" for develop) are used without prompting.
// Any branch that cannot be inferred triggers an interactive prompt.
// Set includeRemote=true to also show remote tracking branches (origin/*)
// in the selection list (useful after a bare clone).
func ResolveBranches(projectRoot string, includeRemote bool) (*BranchSet, error) {
	branches, err := listBranches(projectRoot, includeRemote)
	if err != nil {
		return nil, err
	}

	bs := &BranchSet{}

	// Auto-detect master (prefer "master", fall back to "main").
	bs.Master = findBranch(branches, "master", "main")
	if bs.Master == "" {
		if len(branches) == 0 {
			return nil, fmt.Errorf("no branches found in repository")
		}
		if err := huh.NewSelect[string]().
			Title("Select the branch to serve as master (permanent, production)").
			Options(branchOptions(branches)...).
			Value(&bs.Master).
			Run(); err != nil {
			return nil, fmt.Errorf("selecting master branch: %w", err)
		}
	}

	// Auto-detect develop.
	bs.Develop = findBranch(branches, "develop")
	if bs.Develop == "" {
		const createKey = "__create__"
		opts := []huh.Option[string]{
			huh.NewOption(fmt.Sprintf("Create 'develop' from '%s'", bs.Master), createKey),
		}
		opts = append(opts, branchOptions(branches)...)

		var choice string
		if err := huh.NewSelect[string]().
			Title("'develop' branch not found — pick an existing branch or create it").
			Options(opts...).
			Value(&choice).
			Run(); err != nil {
			return nil, fmt.Errorf("selecting develop branch: %w", err)
		}

		if choice == createKey {
			if _, err := git.Cmd(projectRoot, "branch", "develop", bs.Master); err != nil {
				return nil, fmt.Errorf("creating develop from %s: %w", bs.Master, err)
			}
			bs.Develop = "develop"
		} else {
			bs.Develop = choice
		}
	}

	return bs, nil
}

func listBranches(projectRoot string, includeRemote bool) ([]string, error) {
	out, err := git.Cmd(projectRoot, "branch", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}
	branches := splitLines(out)

	if includeRemote {
		remoteOut, err := git.Cmd(projectRoot, "branch", "-r", "--format=%(refname:short)")
		if err == nil {
			for _, r := range splitLines(remoteOut) {
				// Strip "origin/" prefix; skip the symbolic HEAD pointer.
				name := strings.TrimPrefix(r, "origin/")
				if name != "HEAD" && !containsStr(branches, name) {
					branches = append(branches, name)
				}
			}
		}
	}

	return branches, nil
}

func findBranch(branches []string, candidates ...string) string {
	for _, c := range candidates {
		if containsStr(branches, c) {
			return c
		}
	}
	return ""
}

func branchOptions(branches []string) []huh.Option[string] {
	opts := make([]huh.Option[string], len(branches))
	for i, b := range branches {
		opts[i] = huh.NewOption(b, b)
	}
	return opts
}

func splitLines(s string) []string {
	var lines []string
	for line := range strings.SplitSeq(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func containsStr(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

// WriteBranches persists the master and develop branch names to the local
// git config under [git-vine "branch"]. Called once at init time.
func WriteBranches(projectRoot, master, develop string) error {
	if err := git.SetConfig(projectRoot, "git-vine.branch.master", master); err != nil {
		return fmt.Errorf("saving master branch config: %w", err)
	}
	if err := git.SetConfig(projectRoot, "git-vine.branch.develop", develop); err != nil {
		return fmt.Errorf("saving develop branch config: %w", err)
	}
	return nil
}

// ReadBranches reads the master and develop branch names from the local
// git config. Returns a clear error if the project has not been initialised.
func ReadBranches(projectRoot string) (*BranchSet, error) {
	master, err := git.GetConfig(projectRoot, "git-vine.branch.master")
	if err != nil {
		return nil, fmt.Errorf("project not initialised \u2014 run 'git-vine init' first: %w", err)
	}
	develop, err := git.GetConfig(projectRoot, "git-vine.branch.develop")
	if err != nil {
		return nil, fmt.Errorf("develop branch not configured \u2014 run 'git-vine init' first: %w", err)
	}
	return &BranchSet{Master: master, Develop: develop}, nil
}
