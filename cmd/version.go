package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X git-vine/cmd.Version=<value>".
// Falls back to "dev" when running outside the build script (e.g. go run).
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the git-vine version",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("git-vine", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
