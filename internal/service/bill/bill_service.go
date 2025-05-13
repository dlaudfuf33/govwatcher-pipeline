package bill

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	billAPI "gwatch-data-pipeline/internal/api/bill"
	polticianAPI "gwatch-data-pipeline/internal/api/politician"
	"gwatch-data-pipeline/internal/api/repository"
	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/logging"
	"gwatch-data-pipeline/internal/model/bill"
)

type ImportStats struct {
	TotalFetched  int64
	ProcessedOK   int64
	ProcessedFail int64
}

// 현재 대수 국회의원발의법안 업데이트
func UpdateCurrentBills() {
	apiKey := util.GetNA()

	// 현재 대수 가져오기
	currentAge, err := GetCurrentUnitFromAPI(apiKey)
	if err != nil {
		logging.Errorf("UpdateCurrentBills failed %v", err)
	}

	// 전체 법안 수 가져오기
	totalCount, err := billAPI.FetchTotalBillCount(apiKey, strconv.Itoa(currentAge))
	if err != nil {
		logging.Errorf("UpdateCurrentBills failed %v", err)
	}

	// 총 페이지 수 계산
	totalPages := int(math.Ceil(float64(totalCount) / 100.0))

	// 법안 데이터 수집
	stats, err := ImportBills(apiKey, strconv.Itoa(currentAge), totalPages, 100, 5, 30)
	if err != nil {
		logging.Errorf("UpdateCurrentBills failed %v", err)
	}
	logging.Debugf("UpdateCurrentBills %v", stats)
}

func UpdateCurrentBillsHttp(apiKey string, result chan<- string) {
	// 현재 대수 가져오기
	currentAge, err := GetCurrentUnitFromAPI(apiKey)
	if err != nil {
		logging.Errorf("Failed to fetch current unit: %v", err)
		result <- fmt.Sprintf("Failed to fetch current unit: %v", err)
		return
	}

	// 전체 법안 수 가져오기
	totalCount, err := billAPI.FetchTotalBillCount(apiKey, strconv.Itoa(currentAge))
	if err != nil {
		logging.Errorf("Failed to fetch total count for age=%d: %v", currentAge, err)
		result <- fmt.Sprintf("Failed to fetch total count for age=%d: %v", currentAge, err)
		return
	}

	// 총 페이지 수 계산
	totalPages := int(math.Ceil(float64(totalCount) / 100.0))

	// 법안 데이터 수집
	stats, err := ImportBills(apiKey, strconv.Itoa(currentAge), totalPages, 100, 5, 30)
	if err != nil {
		result <- fmt.Sprintf("Error importing bills for age=%d: %v", currentAge, err)
		return
	}
	// 완료 메시지 전송
	result <- fmt.Sprintf("Age %s: %d bills processed, %d failed", strconv.Itoa(currentAge), stats.ProcessedOK, stats.ProcessedFail)
}

// 국회의원발의법안
func ImportAllBills() {
	apiKey := util.GetNA()

	// 현재 대수 가져오기
	currentUnit, err := GetCurrentUnitFromAPI(apiKey)
	if err != nil {
		logging.Errorf("Failed to fetch current unit: %v", err)
		return
	}

	// 병렬로 각 세대에 대해 ImportBills 호출
	var wg sync.WaitGroup
	for i := 1; i <= currentUnit; i++ {
		wg.Add(1)
		go func(age string) {
			defer wg.Done()
			stats, err := ImportBills(apiKey, age, 100, 100, 5, 30)
			if err != nil {
				logging.Errorf("Error importing bills for age=%s: %v", age, err)
				return
			}
			logging.Infof("Age %s: %d bills processed, %d failed", age, stats.ProcessedOK, stats.ProcessedFail)
		}(strconv.Itoa(i))
	}
	wg.Wait()
}

func ImportBills(apiKey string, age string, maxPage int, pageSize int, apiWorkers int, dbWorkers int) (*ImportStats, error) {
	var stats ImportStats

	pageCh := make(chan int, maxPage)
	billRowCh := make(chan bill.BillRaw, 5000)

	// API Worker
	var apiWg sync.WaitGroup
	for i := 0; i < apiWorkers; i++ {
		apiWg.Add(1)
		go func(workerID int) {
			defer apiWg.Done()
			for page := range pageCh {
				logging.Debugf("[API worker=%d] Fetching AGE=%s page=%d", workerID, age, page)
				rows, err := billAPI.FetchBillList(apiKey, age, page, pageSize)
				if err != nil {
					logging.Warnf("[API worker=%d] error fetching page %d: %v", workerID, page, err)
					continue
				}
				atomic.AddInt64(&stats.TotalFetched, int64(len(rows)))
				for _, r := range rows {
					billRowCh <- r
				}
			}
		}(i)
	}

	// DB Worker
	var dbWg sync.WaitGroup
	for i := 0; i < dbWorkers; i++ {
		dbWg.Add(1)
		go func(workerID int) {
			defer dbWg.Done()
			for r := range billRowCh {
				logging.Debugf("[DB worker=%d] Processing bill %s", workerID, r.BillID)
				ageNum, _ := strconv.Atoi(age)
				err := processBillRowWithError(r, ageNum)
				if err != nil {
					// 실패한 항목 카운트
					atomic.AddInt64(&stats.ProcessedFail, 1)
					logging.Errorf("[DB worker=%d] Failed to process bill %s: %v", workerID, r.BillID, err)
				} else {
					atomic.AddInt64(&stats.ProcessedOK, 1)
				}
			}
		}(i)
	}

	// 페이지 전송
	for page := 1; page <= maxPage; page++ {
		pageCh <- page
	}
	close(pageCh)

	// API 워커 종료되면 billRowCh 닫기
	go func() {
		apiWg.Wait()
		close(billRowCh)
	}()

	dbWg.Wait()

	// 성공 항목 계산: 총 처리된 항목 - 실패 항목
	totalProcessed := stats.TotalFetched - stats.ProcessedFail

	// 로스율 계산
	lossRate := float64(stats.ProcessedFail) / float64(stats.TotalFetched) * 100

	logging.Infof("📊 [AGE=%s] Processed %d bills successfully, %d bills failed ⚖️ Loss rate: %.2f%%", age, totalProcessed, stats.ProcessedFail, lossRate)

	return &stats, nil
}

func parseStepLog(stepLog string) []string {
	if stepLog == "" {
		return []string{}
	}
	return SplitAndTrim(stepLog, ">")
}

// 헬퍼 함수: "a > b > c" → [a, b, c]
func SplitAndTrim(s string, sep string) []string {
	raw := strings.Split(s, sep)
	var out []string
	for _, part := range raw {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func processBillRowWithError(r bill.BillRaw, age int) error {
	// 패닉 처리
	defer func() {
		if r := recover(); r != nil {
			logging.Errorf("panic occurred while processing bill_id=%s: %v", r, r)
		}
	}()
	// processBillRow 호출
	err := processBillRow(r, age)
	if err != nil {
		// 에러를 기록하지만 처리 중단하지 않고 계속 진행
		logging.Errorf("Error processing bill_id=%s: %v", r.BillID, err)
	}
	return err
}

func processBillRow(r bill.BillRaw, age int) error {
	logging.Infof("📄 Processing bill: %s (%s)", r.BillID, r.Title)

	summary := ""
	stepLog := ""
	currentStep := ""

	// 1. 상세 페이지 존재 여부 확인
	if strings.TrimSpace(r.DetailLink) != "" {
		var err error
		summary, stepLog, currentStep, err = billAPI.FetchBillDetailInfo(r.DetailLink)
		if err != nil {
			logging.Warnf("Failed to fetch detail info (bill_id=%s): %v", r.BillID, err)
			return err
		}
	} else {
		logging.Warnf("No DetailLink for bill_id=%s, skipping detail fetch", r.BillID)
	}
	committeeID, err := repository.GetOrCreateCommittee(db.DB, r.Committee)
	if err != nil {
		logging.Errorf("Committee lookup failed: %v", err)
		committeeID = 0
	}

	// 2. 모델 변환
	billEntity := r.ToEntity(summary, currentStep, committeeID)

	// Age는 외부에서 전달받은 파라미터로 직접 설정
	billEntity.Age = age

	// 3. billEntity 먼저 저장
	upsertBill(&billEntity)

	// 4. billEntity.ID를 BillStatusFlow에 채워서 생성
	var statusFlows []bill.BillStatusFlow
	for idx, step := range parseStepLog(stepLog) {
		if step == "" {
			continue
		}
		statusFlows = append(statusFlows, bill.BillStatusFlow{
			BillID:    billEntity.ID,
			StepOrder: idx + 1,
			StepName:  step,
		})
	}

	// 5. statusFlows 저장
	for _, flow := range statusFlows {
		upsertBillStep(&flow)
	}

	// 6. 제안자 크롤링 및 저장
	if r.MemberListURL != "" {
		relations, err := billAPI.FetchAndMatchProposers(billEntity.ID, r.MemberListURL, age)
		if err != nil {
			logging.Warnf("failed to match proposers: %v", err)
			return err
		}
		for _, rel := range relations {
			upsertRelation(&rel)
		}
	}

	return nil
}

// Upsert
func upsertBill(b *bill.Bill) {
	res := db.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "bill_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"result":       gorm.Expr("CASE WHEN bills.result IS DISTINCT FROM excluded.result THEN excluded.result ELSE bills.result END"),
			"current_step": gorm.Expr("CASE WHEN bills.current_step IS DISTINCT FROM excluded.current_step THEN excluded.current_step ELSE bills.current_step END"),
			"updated_at":   gorm.Expr("NOW()"),
		}),
	}).Create(b)

	if res.Error != nil {
		logging.Errorf("DB insert error: %v", res.Error)
	}
}

func upsertBillStep(s *bill.BillStatusFlow) {
	res := db.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "bill_id"},
			{Name: "step_order"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"step_name":  gorm.Expr("CASE WHEN bill_status_flows.step_name IS DISTINCT FROM excluded.step_name THEN excluded.step_name ELSE bill_status_flows.step_name END"),
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(s)

	if res.Error != nil {
		logging.Errorf("DB insert error: %v", res.Error)
	}
}

func upsertRelation(r *bill.BillPoliticianRelation) {
	res := db.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "bill_id"},
			{Name: "politician_id"},
			{Name: "role"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"role":       gorm.Expr("CASE WHEN bill_politician_relations.role IS DISTINCT FROM excluded.role THEN excluded.role ELSE bill_politician_relations.role END"),
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(r)

	if res.Error != nil {
		logging.Errorf("DB insert error: %v", res.Error)
	}
}
func GetCurrentUnitFromAPI(apiKey string) (int, error) {
	// 페이지 크기 설정
	pageSize := 1
	page := 1

	// 현재 대수의 정보를 가져오기 위해 FetchCurrentPoliticians 호출
	politicians, err := polticianAPI.FetchCurrentPoliticians(apiKey, page, pageSize)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch current politicians: %v", err)
	}

	if len(politicians) == 0 {
		return 0, fmt.Errorf("no current politicians found")
	}

	// "제22대"와 같은 문자열에서 숫자만 추출
	re := regexp.MustCompile(`\d+`)                // 숫자만 추출하는 정규 표현식
	unitStr := re.FindString(politicians[0].Units) // "22" 추출

	if unitStr == "" {
		return 0, fmt.Errorf("failed to extract unit number from: %v", politicians[0].Units)
	}

	// 문자열을 정수로 변환
	currentUnit, err := strconv.Atoi(unitStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert current unit to int: %v", err)
	}

	return currentUnit, nil
}
