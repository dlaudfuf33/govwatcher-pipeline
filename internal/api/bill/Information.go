package bill

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/logging"
	model "gwatch-data-pipeline/internal/model/bill"
)

var apiKey = os.Getenv("NA_KEY")

type openAPIBillResponse struct {
	AllBill []struct {
		Row []struct {
			BillID    string `json:"BILL_ID"`
			BillNo    string `json:"BILL_NO"`
			BillName  string `json:"BILL_NM"`
			ProposeDt string `json:"PPSL_DT"`
		} `json:"row"`
	} `json:"ALLBILL"`
}

// bill_no로 bill_id 못 찾는 경우 OpenAPI에서 조회 후 bills 테이블에 삽입하는 함수
func FetchAndInsertBillFromOpenAPI(billNo string, db *gorm.DB) (*model.Bill, error) {
	url := fmt.Sprintf("https://open.assembly.go.kr/portal/openapi/ALLBILL?KEY=%s&Type=json&pIndex=1&pSize=5&BILL_NO=%s", apiKey, billNo)
	logging.Debugf("🔎 Calling OpenAPI for bill_no=%s", billNo)

	resp, err := util.MakeRequestWithUA("GET", url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed openAPIBillResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	if len(parsed.AllBill) < 2 || len(parsed.AllBill[1].Row) == 0 {
		return nil, fmt.Errorf("no bill found from OpenAPI")
	}

	item := parsed.AllBill[1].Row[0]

	newBill := model.Bill{
		BillID:      strings.TrimSpace(item.BillID),
		BillNo:      billNo,
		Title:       strings.TrimSpace(item.BillName),
		ProposeDate: parseDate(strings.TrimSpace(item.ProposeDt)),
	}

	if err := db.Create(&newBill).Error; err != nil {
		return nil, err
	}

	return &newBill, nil
}

func parseDate(raw string) *time.Time {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil
	}
	return &t
}
