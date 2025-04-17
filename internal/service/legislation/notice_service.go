package legislation

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	billAPI "gwatch-data-pipeline/internal/api/bill"
	client "gwatch-data-pipeline/internal/api/legislation"
	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/logging"
	model "gwatch-data-pipeline/internal/model/legislation"
)

type BillInfo struct {
	BillNo       string
	CommentCount int
}

// 경로에서 파일 가져와서 처리하는 함수
func ImportNoticePeriodsFromList(db *gorm.DB) error {
	filePath := util.GetLatestNoticeFilePath()
	logging.Infof("📄 Opening Excel file: %s", filePath)
	defer os.Remove(filePath)

	billNos, err := ReadBillNosFromExcel(filePath)
	if err != nil {
		logging.Errorf("Failed to read bill numbers from Excel: %v", err)
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(billNos))

	for _, bill := range billNos {
		wg.Add(1)
		bill := bill
		go func() {
			defer wg.Done()
			if err := processSingleBill(bill, db); err != nil {
				logging.Errorf("Error processing bill %s: %v", bill.BillNo, err)
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	logging.Infof("Completed fetching notice periods")
	return nil
}

func processSingleBill(bill BillInfo, db *gorm.DB) error {
	startInner := time.Now()
	logging.Infof("🔍 Fetching notice for bill: %s (%d comments)", bill.BillNo, bill.CommentCount)

	billID, err := client.GetBillIDByNo(bill.BillNo, db)
	if err != nil {
		if billID == "" {
			logging.Warnf("Fallback to OpenAPI for bill_no=%s", bill.BillNo)
			billID, err = billAPI.FetchAndInsertBillFromOpenAPI(bill.BillNo, db)
		}
		if err != nil {
			logging.Errorf("Failed to get bill_id via fallback: %v", err)
			return err
		}
	}
	url := fmt.Sprintf("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId=%s&searchConClosed=0", billID)
	noticePeriod, commentsCount, err := client.FetchNoticePeriodFast(url)
	if err != nil {
		logging.Errorf("Failed to fetch notice period: %v", err)
		return err
	}

	err = upsertLegislativeNotice(db, billID, noticePeriod, commentsCount)
	if err != nil {
		logging.Errorf("Failed to update legislative notice: %v", err)
	}
	logging.Infof("⏱️ [ImportNoticePeriodsFromList]:%s took %s", bill.BillNo, time.Since(startInner))
	return nil
}

// Excel 파일에서 bill_no를 읽어서 반환하는 함수
func ReadBillNosFromExcel(filePath string) ([]BillInfo, error) {
	logging.Infof("📄 Opening Excel file: %s", filePath)
	logging.Debugf("Starting to read bill numbers from Excel file: %s", filePath)
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var billNos []BillInfo
	sheetName := f.GetSheetName(0)
	logging.Infof("📄 Detected sheet: %s", sheetName)
	if sheetName == "" {
		logging.Errorf("no sheets found in file")
		return nil, fmt.Errorf("no sheets found in file")
	}
	rows, err := f.GetRows(sheetName)

	for _, row := range rows[1:] {
		if len(row) < 3 {
			continue
		}
		billNos = append(billNos, BillInfo{
			BillNo:       row[1],
			CommentCount: parseCommentCount(row[2]),
		})
	}
	return billNos, nil
}

func parseCommentCount(str string) int {
	n, err := strconv.Atoi(strings.TrimSpace(str))
	if err != nil {
		logging.Debugf("Invalid comment count: %s", str)
		return 0
	}
	return n
}

// legislative_notice을 추가하거나 업데이트하는 함수
func upsertLegislativeNotice(db *gorm.DB, billID string, noticePeriod string, commentCount int) error {
	var startDate, endDate time.Time
	parts := strings.Split(noticePeriod, "~")
	if len(parts) == 2 {
		var parseErr error
		startDate, parseErr = time.Parse("2006-01-02", strings.TrimSpace(parts[0]))
		if parseErr != nil {
			logging.Errorf("Invalid start date format for bill: %s", parts[0])
			return fmt.Errorf("invalid start date")
		}
		endDate, parseErr = time.Parse("2006-01-02", strings.TrimSpace(parts[1]))
		if parseErr != nil {
			logging.Errorf("Invalid end date format for bill: %s", parts[1])
			return fmt.Errorf("invalid end date")
		}
	} else {
		logging.Errorf("Invalid notice period format for bill: %s", noticePeriod)
		return fmt.Errorf("invalid notice period format")
	}

	notice := model.LegislativeNotice{
		BillID:        billID,
		CommentsCount: commentCount,
		StartDate:     startDate,
		EndDate:       endDate,
		CommentsURL:   fmt.Sprintf("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId=%s&searchConClosed=0", billID),
	}

	logging.Infof("💾 Saving notice to DB for bill_id=%s with start=%v end=%v", notice.BillID, startDate, endDate)

	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "bill_id"}},
		UpdateAll: true,
	}).Create(&notice).Error; err != nil {
		logging.Errorf("Failed to upsert legislative notice: %v", err)
		return err
	}

	logging.Infof("Legislative notice upserted successfully: %s", notice.BillID)
	return nil
}