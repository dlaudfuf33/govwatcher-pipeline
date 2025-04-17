package cmd

import (
	"github.com/spf13/cobra"

	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/service/legislation"
)

var days int

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update opinions within N days",
	Run: func(cmd *cobra.Command, args []string) {
		db.InitDB()
		defer db.CloseDB()
		legislation.ImportOpinionCommentsFromLatestFileWithinDays(db.DB, days)
	},
}

func init() {
	updateCmd.Flags().IntVarP(&days, "days", "d", 1, "Number of days to look back")
	rootCmd.AddCommand(updateCmd)
}