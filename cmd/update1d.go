package cmd

import (
	"github.com/spf13/cobra"

	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/service/legislation"
)

var update1dCmd = &cobra.Command{
	Use:   "update-1d",
	Short: "Update opinions within the past 1 day",
	Run: func(cmd *cobra.Command, args []string) {
		db.InitDB()
		defer db.CloseDB()
		legislation.ImportOpinionCommentsFromLatestFileWithinDays(db.DB, 1)
	},
}

func init() {
	rootCmd.AddCommand(update1dCmd)
}