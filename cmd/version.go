package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of mesa core",
	Long:  `All software has versions. This is mesa's`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Core Version 1.0")
		os.Exit(0)
	},
}
