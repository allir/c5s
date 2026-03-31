package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/allir/c5s/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("c5s %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
