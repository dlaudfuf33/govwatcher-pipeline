package legislation

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gwatch-data-pipeline/internal/api/legislation"
	"gwatch-data-pipeline/internal/logging"
	model "gwatch-data-pipeline/internal/model"
	modelLegislation "gwatch-data-pipeline/internal/model/legislation"
)

// 입법예고 테이블에서 유효한 데이터 조회 후 병렬로 의견 다운로드를 수행하는 진입점 함수
func ImportOpinionCommentsFromLatestFile(db *gorm.DB) error {
	start := time.Now()

	notices, err := getValidNotices(db)
	if err != nil {
		return err
	}
	if len(notices) == 0 {
		return nil
	}
	sampleBillID := notices[0].BillID
	session, err := PrepareSession(sampleBillID)
	if err != nil {
		return err
	}

	var billIDs []string
	for _, n := range notices {
		billIDs = append(billIDs, n.BillID)
	}

	failed := downloadWithWorkers(billIDs, session, 3)
	if len(failed) > 0 {
		logging.Warnf("%d downloads failed, retrying...", len(failed))
		retryFailed := downloadWithWorkers(failed, session, 3)
		if len(retryFailed) > 0 {
			logging.Warnf("%d bills failed even after retry: %v", len(retryFailed), retryFailed)
		}
	}
	logging.Infof("⏱️ [ImportOpinionCommentsFromLatestFile] took %s", time.Since(start))
	return nil
}

func ImportOpinionCommentsFromLatestFileWithinDays(db *gorm.DB,withinDays int) error {
	start := time.Now()

	notices, err := getImminentValidNotices(db,withinDays)
	if err != nil {
		return err
	}
	if len(notices) == 0 {
		return nil
	}
	sampleBillID := notices[0].BillID
	session, err := PrepareSession(sampleBillID)
	if err != nil {
		return err
	}

	var billIDs []string
	for _, n := range notices {
		billIDs = append(billIDs, n.BillID)
	}

	failed := downloadWithWorkers(billIDs, session, 3)
	if len(failed) > 0 {
		logging.Warnf("%d downloads failed, retrying...", len(failed))
		retryFailed := downloadWithWorkers(failed, session, 3)
		if len(retryFailed) > 0 {
			logging.Warnf("%d bills failed even after retry: %v", len(retryFailed), retryFailed)
		}
	}
	logging.Infof("⏱️ [ImportOpinionCommentsFromLatestFile] took %s", time.Since(start))
	return nil
}


// DB에서 현재 시각 기준 유효한 입법예고 데이터를 조회하는 함수
func getValidNotices(db *gorm.DB) ([]modelLegislation.LegislativeNotice, error) {
	var notices []modelLegislation.LegislativeNotice
	now := time.Now()
	if err := db.Where("end_date >= ?", now).Find(&notices).Error; err != nil {
		return nil, fmt.Errorf("failed to query valid legislative notices: %v", err)
	}
	return notices, nil
}

// DB에서 현재 시각 기준 종료 N일 남은 입법예고 데이터를 조회하는 함수
func getImminentValidNotices(db *gorm.DB,withinDays int) ([]modelLegislation.LegislativeNotice, error) {
	var notices []modelLegislation.LegislativeNotice
	now := time.Now()
	until := now.AddDate(0, 0, withinDays)
	if err := db.Where("end_date >= ? AND end_date <= ?", now, until).
	Find(&notices).Error; err != nil {
		return nil, fmt.Errorf("failed to query valid legislative notices: %v", err)
	}
	return notices, nil
}


// DB에서 현재 시각 기준 유효한 입법예고 ID 조회하는 함수
func GetValidNoticeID(db *gorm.DB) (LegislativeNoticeID string, err error) {
	var notice modelLegislation.LegislativeNotice
	now := time.Now()
	if err := db.Where("end_date >= ? limit 1", now).Find(&notice).Error; err != nil {
		return "", fmt.Errorf("failed to query valid legislative notices: %v", err)
	}
	return notice.BillID, nil
}

// chromedp 세션을 초기화하고 CSRF 토큰 및 쿠키를 수집하는 함수
func PrepareSession(billID string) (model.SessionInfo, error) {
	ctx, cancel := legislation.CreateChromedpContext()

	// 1. warm-up → 필수!
	if err := legislation.WarmUpSessionWithViewPage(ctx, billID); err != nil {
		cancel()
		return model.SessionInfo{}, fmt.Errorf("Failed to warm up session: %v", err)
	}

	// 2. 쿠키 추출 
	cookies, err := legislation.GetCookiesForRequest(ctx)
	if err != nil {
		cancel()
		return model.SessionInfo{}, fmt.Errorf("Failed to retrieve cookies: %v", err)
	}

	// 3. CSRF 토큰 추출 (view 페이지에서)
	viewURL := "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId=" + billID + "&searchConClosed=0"
	csrfToken, err := legislation.FetchCSRFToken(ctx, viewURL)
	if err != nil {
		cancel()
		return model.SessionInfo{}, fmt.Errorf("Failed to fetch CSRF token: %v", err)
	}

	return model.SessionInfo{
		Ctx:       ctx,
		Cancel:    cancel,
		CSRFToken: csrfToken,
		Cookies:   cookies,
	}, nil
}

// 워커풀을 구성하여 각 billID에 대해 의견 엑셀 파일을 병렬 다운로드하는 함수
func downloadWithWorkers(billIDs []string, session model.SessionInfo, maxWorkers int) []string {
	jobs := make(chan string, len(billIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for billID := range jobs {
				err := legislation.DownloadOpinionXlsxWithSession(session, billID)
				if err != nil {
					logging.Errorf("Worker %d: Failed to download opinion for bill %s: %v", workerID, billID, err)
					mu.Lock()
					failed = append(failed, billID)
					mu.Unlock()
				}
			}
		}(i)
	}

	for _, billID := range billIDs {
		jobs <- billID
	}
	close(jobs)
	wg.Wait()
	return failed
}

func ParseAndInsertOpinionsFromDownloads(db *gorm.DB) {
	opinionWorkers := 10;
	tempBillID, err := GetValidNoticeID(db)

	session, err := PrepareSession(tempBillID)
	if err != nil {
		logging.Errorf("failed prepareSession %v:", err)
		return 
	}

	files, err := filepath.Glob("downloads/opinion/*.xlsx")
	if err != nil {
		logging.Errorf("failed to list files: %v", err)
		return
	}

	for _, file := range files {
		f, err := excelize.OpenFile(file)
		if err != nil {
			logging.Errorf("failed to open file %s: %v", file, err)
			return
		}

		sheetName := f.GetSheetName(0)
		logging.Infof("📄 Detected sheet: %s", sheetName)
		if sheetName == "" {
			logging.Errorf("no sheets found in file")
			return
		}

		rows, err := f.GetRows(sheetName)
		if err != nil {
			logging.Errorf("failed to get rows from file %s: %v", file, err)
			return
		}

		type job struct {
			billID     string
			opnNo      string
			subject    string
			author     string
			createdAt  string
		}

		var wg sync.WaitGroup
		jobs := make(chan job, len(rows)-1)

		for i := 0; i < opinionWorkers; i++ {
			wg.Add(1)
			workerID := i
			go func(id int) {
				defer wg.Done()
				for j := range jobs {
					logging.Debugf("👨🏻‍🔧 Worker %d processing opinion %s", id, j.opnNo)

					parsedAt, err := time.Parse("2006-01-02", j.createdAt)
					if err != nil {
						logging.Warnf("Invalid createdAt format for %s: %s", j.opnNo, j.createdAt)
						parsedAt = time.Now()
					}

					isAnonymous := inferAnonymous(j.subject, "")
					content := ""
					if isAnonymous == nil || !*isAnonymous {
						content, err = legislation.FetchOpinionContent(j.billID, j.opnNo, session)
						if err != nil {
							logging.Errorf("Worker %d: Failed to fetch content for opinion number %s: %v", id, j.opnNo, err)
							return
						}
					}

					agreement := inferAgreement(j.subject, content)

					opnID, err := strconv.ParseUint(j.opnNo, 10, 64)
					if err != nil {
						logging.Warnf("Invalid OpnNo format: %s", j.opnNo)
						return
					}

					if err := db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&modelLegislation.LegislativeOpinion{
						OpnNo:       opnID,
						BillID:      j.billID,
						Subject:     j.subject,
						Author:      j.author,
						Content:     content,
						CreatedAt:   parsedAt,
						Agreement:   agreement,
						IsAnonymous: isAnonymous,
					}).Error; err != nil {
						logging.Errorf("Worker %d: Failed to insert/update opinion for %s: %v", id, j.opnNo, err)
					}
				}
			}(workerID)
		}

		base := filepath.Base(file)
		billID := strings.Split(base, ",")[0]
		maxOpnNo, err := GetMaxOpnNoByBillID(db, billID)
		if err != nil {
			logging.Errorf("Failed to get max opnNo for %s: %v", billID, err)
			maxOpnNo = 0
		}
		for _, row := range rows[1:] {
			if len(row) < 6 {
				continue
			}
			opnNoParsed, err := strconv.ParseUint(row[1], 10, 64)
			if err != nil {
				logging.Errorf("strconv.ParseUint failed : %v",err)
				continue
			}
			if opnNoParsed <= maxOpnNo {
				continue
			}

			jobs <- job{
				billID:    billID,
				opnNo:     row[1],
				subject:   row[2],
				author:    row[3],
				createdAt: row[5],
			}
		}

		close(jobs)
		wg.Wait()

		if err := os.Remove(file); err != nil {
			logging.Errorf("Failed to delete file %s: %v", file, err)
		}
	}
	return
}

func inferAgreement(subject, content string) *bool {
	s := strings.ToLower(subject + " " + content)
	pos := strings.Count(s, "찬성")
	neg := strings.Count(s, "반대")
	if pos == 0 && neg == 0 {
		return nil
	}
	if pos > neg {
		v := true
		return &v
	}
	if neg > pos {
		v := false
		return &v
	}
	return nil
}

func inferAnonymous(subject, content string) *bool {
	s := strings.ToLower(subject + " " + content)
	if strings.Contains(s, "[비공개]") {
		v := true
		return &v
	}else{
		v := false
		return &v
	}
}

func GetMaxOpnNoByBillID(db *gorm.DB, billID string) (uint64, error) {
	var maxOpnNo uint64
	err := db.Model(&modelLegislation.LegislativeOpinion{}).
		Where("bill_id = ?", billID).
		Select("MAX(opn_no)").
		Scan(&maxOpnNo).Error
	return maxOpnNo, err
}
