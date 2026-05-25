package cmd

import "github.com/spf13/cobra"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a git-wtf project",
	Long: `Initialize a git-wtf project using one of two modes:

  fresh  — create a new repository from scratch
  adopt  — convert an existing local git clone in-place`,
}

func init() {
	rootCmd.AddCommand(initCmd)
}
