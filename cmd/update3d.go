package cmd

import (
	"github.com/spf13/cobra"

	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/service/legislation"
)

var update3dCmd = &cobra.Command{
	Use:   "update-3d",
	Short: "Update opinions within the past 3 days",
	Run: func(cmd *cobra.Command, args []string) {
		db.InitDB()
		defer db.CloseDB()
		legislation.ImportOpinionCommentsFromLatestFileWithinDays(db.DB, 3)
	},
}

func init() {
	rootCmd.AddCommand(update3dCmd)
}