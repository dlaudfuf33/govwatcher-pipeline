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

// 🧹 유효한 입법예고 조회 후 병렬로 의견 다운로드
func ImportOpinionCommentsFromLatestFile(db *gorm.DB) error {
	start := time.Now()

	notices, err := getValidNotices(db)
	if err != nil {
		return err
	}
	if len(notices) == 0 {
		return nil
	}
	var billID string
	err = db.Raw("SELECT bill_id FROM bills WHERE id = ?", notices[0].BillID).Scan(&billID).Error
	if err != nil || billID == "" {
		return fmt.Errorf("failed to find bill_id for notice id=%d: %v", notices[0].BillID, err)
	}
	session, err := PrepareSession(billID)
	if err != nil {
		return err
	}

	var billIDs []string
	for _, n := range notices {
		var billID string
		err := db.Raw("SELECT bill_id FROM bills WHERE id = ?", n.BillID).Scan(&billID).Error
		if err != nil || billID == "" {
			logging.Warnf("Skipping notice id=%d: failed to find bill_id: %v", n.BillID, err)
			continue
		}
		billIDs = append(billIDs, billID)
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

// 🧹 N일 이내 유효 입법예고 조회 후 병렬 의견 다운로드
func ImportOpinionCommentsFromLatestFileWithinDays(db *gorm.DB, withinDays int) error {
	start := time.Now()

	notices, err := getImminentValidNotices(db, withinDays)
	if err != nil {
		return err
	}
	if len(notices) == 0 {
		return nil
	}

	var billID string
	err = db.Raw("SELECT bill_id FROM bills WHERE id = ?", notices[0].BillID).Scan(&billID).Error
	if err != nil || billID == "" {
		return fmt.Errorf("failed to find bill_id for notice id=%d: %v", notices[0].BillID, err)
	}
	session, err := PrepareSession(billID)
	if err != nil {
		return err
	}

	var billIDs []string
	for _, n := range notices {
		var billID string
		err := db.Raw("SELECT bill_id FROM bills WHERE id = ?", n.BillID).Scan(&billID).Error
		if err != nil || billID == "" {
			logging.Warnf("Skipping notice id=%d: failed to find bill_id: %v", n.BillID, err)
			continue
		}
		billIDs = append(billIDs, billID)
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

// 🧹 DB에서 현재 시각 기준 유효 입법예고 조회
func getValidNotices(db *gorm.DB) ([]modelLegislation.LegislativeNotice, error) {
	var notices []modelLegislation.LegislativeNotice
	now := time.Now()
	if err := db.Where("end_date >= ?", now).Find(&notices).Error; err != nil {
		return nil, fmt.Errorf("failed to query valid legislative notices: %v", err)
	}
	return notices, nil
}

// 🧹 DB에서 종료 N일 이내 유효 입법예고 조회
func getImminentValidNotices(db *gorm.DB, withinDays int) ([]modelLegislation.LegislativeNotice, error) {
	var notices []modelLegislation.LegislativeNotice
	now := time.Now()
	until := now.AddDate(0, 0, withinDays)
	if err := db.Where("end_date >= ? AND end_date <= ?", now, until).
		Find(&notices).Error; err != nil {
		return nil, fmt.Errorf("failed to query valid legislative notices: %v", err)
	}
	return notices, nil
}

// 🧹 DB에서 현재 시각 기준 유효 입법예고 ID 조회
func GetValidNoticeID(db *gorm.DB) (LegislativeNoticeID string, err error) {
	var notice modelLegislation.LegislativeNotice
	now := time.Now()
	if err := db.Where("end_date >= ?", now).Order("end_date ASC").Limit(1).Find(&notice).Error; err != nil {
		return "", fmt.Errorf("failed to query valid legislative notices: %v", err)
	}

	var billID string
	err = db.Raw("SELECT bill_id FROM bills WHERE id = ?", notice.BillID).Scan(&billID).Error
	if err != nil || billID == "" {
		return "", fmt.Errorf("failed to fetch associated bill_id for notice id=%d: %v", notice.BillID, err)
	}
	return billID, nil
}

// 🔥 세션 준비 (쿠키 + 토큰)
func PrepareSession(billID string) (model.SessionInfo, error) {
	ctx, cancel := legislation.CreateChromedpContext()

	// 1. warm-up → 필수! 🔥
	if err := legislation.WarmUpSessionWithViewPage(ctx, billID); err != nil {
		cancel()
		return model.SessionInfo{}, fmt.Errorf("Failed to warm up session: %v", err)
	}

	// 2. 쿠키 추출 🍪
	cookies, err := legislation.GetCookiesForRequest(ctx)
	if err != nil {
		cancel()
		return model.SessionInfo{}, fmt.Errorf("Failed to retrieve cookies: %v", err)
	}

	// 3. CSRF 토큰 추출 (view 페이지에서) 🔐
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

// 🛠️ 워커풀로 병렬 의견 엑셀 다운로드
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

// 📥 다운로드된 의견 파일 읽고 병렬 DB 저장
func ParseAndInsertOpinionsFromDownloads(db *gorm.DB) {
	opinionWorkers := 20
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
			billID    string
			opnNo     string
			subject   string
			author    string
			createdAt string
		}

		// 🗂️ 파일명에서 bill_id 추출
		base := filepath.Base(file)
		billID := strings.Split(base, ",")[0]

		// 🔍 bill_no로 bill_id 조회
		var bID uint64
		err = db.Raw("SELECT id FROM bills WHERE bill_id = ?", billID).Scan(&bID).Error
		if err != nil || billID == "" {
			logging.Warnf("Skipping file %s: failed to find bills.id for billID %s: %v", base, billID, err)
			continue
		}

		// 🔍 bills.id로 notice_id 조회
		var noticeID uint64
		err = db.Raw("SELECT id FROM legislative_notices WHERE bill_id = ?", bID).Scan(&noticeID).Error
		if err != nil {
			logging.Warnf("Skipping file %s: failed to find legislative_notice id for bill_id %d: %v", base, bID, err)
			continue
		}

		maxOpnNo, err := GetMaxOpnNoByNoticeID(db, noticeID)
		if err != nil {
			logging.Errorf("Failed to get max opnNo for noticeID %d: %v", noticeID, err)
			maxOpnNo = 0
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

					isAnonymous := inferAnonymous(j.subject, "")
					content := ""
					parsedCreatedAt, err := time.Parse("2006-01-02", j.createdAt)

					if isAnonymous != nil && !*isAnonymous {
						contentFetched, fetchedCreatedAtStr, err := legislation.FetchOpinionContent(j.billID, j.opnNo, session)
						if err != nil {
							logging.Errorf("Worker %d: Failed to fetch content for opinion number %s: %v", id, j.opnNo, err)
							continue
						}
						content = contentFetched
						parsedCreatedAt = fetchedCreatedAtStr
					}

					agreement := inferAgreement(j.subject, content)
					enumVal := DetermineAgreementEnum(isAnonymous, agreement)
					opnID, err := strconv.ParseUint(j.opnNo, 10, 64)
					if err != nil {
						logging.Warnf("Invalid OpnNo format: %s", j.opnNo)
						return
					}

					if err := db.Clauses(clause.OnConflict{
						Columns:   []clause.Column{{Name: "notice_id"}, {Name: "opn_no"}},
						DoUpdates: clause.AssignmentColumns([]string{"subject", "author", "content", "created_at", "agreement"}),
					}).Create(&modelLegislation.LegislativeOpinion{
						OpnNo:     opnID,
						NoticeID:  noticeID,
						Subject:   j.subject,
						Author:    j.author,
						Content:   content,
						CreatedAt: parsedCreatedAt,
						Agreement: enumVal,
					}).Error; err != nil {
						logging.Errorf("Worker %d: Failed to insert/update opinion for %s: %v", id, j.opnNo, err)
					}
				}
			}(workerID)
		}

		for _, row := range rows[1:] {
			if len(row) < 6 {
				continue
			}
			opnNoParsed, err := strconv.ParseUint(row[1], 10, 64)
			if err != nil {
				logging.Errorf("strconv.ParseUint failed : %v", err)
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

// 🔍 의견 찬반 여부 추론
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

// 🔍 의견 익명 여부 추론
func inferAnonymous(subject, content string) *bool {
	s := strings.ToLower(subject + " " + content)
	if strings.Contains(s, "[비공개]") {
		v := true
		return &v
	} else {
		v := false
		return &v
	}
}

// 🧹 bill_id 기준 최대 의견 번호 조회
func GetMaxOpnNoByBillID(db *gorm.DB, billID string) (uint64, error) {
	var maxOpnNo uint64
	err := db.Model(&modelLegislation.LegislativeOpinion{}).
		Where("bill_id = ?", billID).
		Select("MAX(opn_no)").
		Scan(&maxOpnNo).Error
	return maxOpnNo, err
}

// 🧹 notice_id 기준 최대 의견 번호 조회
func GetMaxOpnNoByNoticeID(db *gorm.DB, noticeID uint64) (uint64, error) {
	var maxOpnNo uint64
	err := db.Model(&modelLegislation.LegislativeOpinion{}).
		Where("notice_id = ?", noticeID).
		Select("MAX(opn_no)").
		Scan(&maxOpnNo).Error
	return maxOpnNo, err
}

const (
	AgreementPrivate  = "PRIVATE"
	AgreementAgree    = "AGREE"
	AgreementDisagree = "DISAGREE"
)

func DetermineAgreementEnum(isAnonymous *bool, isAgree *bool) string {
	if isAnonymous != nil && *isAnonymous {
		return AgreementPrivate
	}
	if isAgree != nil && *isAgree {
		return AgreementAgree
	}
	return AgreementDisagree
}
