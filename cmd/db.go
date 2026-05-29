package cmd

import (
	"app/db"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCommand)
}

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "migrates the core database",
	Run: func(cmd *cobra.Command, args []string) {
		if err := db.MigrateDB(false); err != nil {
			panic(err)
		}
		os.Exit(0)
	},
}
