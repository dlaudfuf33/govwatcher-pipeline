package cmd

import (
	"github.com/spf13/cobra"

	legislationAPI "gwatch-data-pipeline/internal/api/legislation"
	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/service/bill"
	"gwatch-data-pipeline/internal/service/legislation"
	"gwatch-data-pipeline/internal/service/poltician"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize full dataset",
	Run: func(cmd *cobra.Command, args []string) {
		db.InitDB()
		defer db.CloseDB()

		poltician.ImportAllPoliticians()
		bill.ImportAllBills()
		legislationAPI.DownloadLegislativeListXlsx()
		legislation.ImportNoticePeriodsFromList(db.DB)
		legislation.ImportOpinionCommentsFromLatestFile(db.DB)
		legislation.ParseAndInsertOpinionsFromDownloads(db.DB)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
