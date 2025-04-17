package cmd

import (
	"github.com/spf13/cobra"

	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/service/legislation"
)

var update7dCmd = &cobra.Command{
	Use:   "update-7d",
	Short: "Update opinions within the past 7 days",
	Run: func(cmd *cobra.Command, args []string) {
		db.InitDB()
		defer db.CloseDB()
		legislation.ImportOpinionCommentsFromLatestFileWithinDays(db.DB, 7)
	},
}

func init() {
	rootCmd.AddCommand(update7dCmd)
}