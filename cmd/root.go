// Package cmd wires the Cobra command tree for git-wtf.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "git-wtf",
	Short: "Git worktree flow manager",
	Long: `git-wtf manages Git worktrees with a strict Git Flow branching model.

Place the binary in your PATH and invoke it as 'git wtf' via Git's
custom-command discovery (git-<name> → git <name>).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the main entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
