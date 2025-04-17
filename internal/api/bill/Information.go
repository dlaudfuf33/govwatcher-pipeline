package bill

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gorm.io/gorm"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/logging"
	model "gwatch-data-pipeline/internal/model/legislation"
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
func FetchAndInsertBillFromOpenAPI(billNo string, db *gorm.DB) (string, error) {
	url := fmt.Sprintf("https://open.assembly.go.kr/portal/openapi/ALLBILL?KEY=%s&Type=json&pIndex=1&pSize=5&BILL_NO=%s", apiKey, billNo)
	logging.Debugf("🔎 Calling OpenAPI for bill_no=%s", billNo)

	resp, err := util.MakeRequestWithUA("GET", url)
	if err != nil {
		logging.Errorf(" Failed to call OpenAPI: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	fmt.Println("내용 < \n", string(body))
	if err != nil {
		logging.Errorf(" Failed to read OpenAPI response: %v", err)
		return "", err
	}

	var parsed openAPIBillResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		logging.Errorf(" Failed to parse OpenAPI JSON: %v", err)
		return "", err
	}

	if len(parsed.AllBill) < 2 || len(parsed.AllBill[1].Row) == 0 {
		logging.Warnf(" No bill found in OpenAPI for bill_no=%s", billNo)
		return "", fmt.Errorf("no bill found from OpenAPI")
	}

	item := parsed.AllBill[1].Row[0]
	billID := strings.TrimSpace(item.BillID)
	name := strings.TrimSpace(item.BillName)
	proposeDate := strings.TrimSpace(item.ProposeDt)

	logging.Infof("📥 Inserting fallback bill from OpenAPI: %s (%s)", billID, name)

	newBill := model.Bill{
		BillID:      billID,
		BillNo:      billNo,
		Name:        name,
		ProposeDate: proposeDate,
	}

	if err := db.Create(&newBill).Error; err != nil {
		logging.Errorf(" Failed to insert bill from OpenAPI: %v", err)
		return "", err
	}

	return billID, nil
}