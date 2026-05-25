package cmd

import "github.com/spf13/cobra"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a git-wtf project",
	Long: `Initialize a git-wtf project using one of three modes:

  fresh  — create a new bare repository from scratch
  adopt  — convert an existing local git clone in-place
  clone  — clone a remote repository and set up the structure`,
}

func init() {
	rootCmd.AddCommand(initCmd)
}
