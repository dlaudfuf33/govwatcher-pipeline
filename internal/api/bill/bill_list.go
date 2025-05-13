package bill

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/logging"
	"gwatch-data-pipeline/internal/model/bill"
)

var ErrNoData = errors.New("no data found")

type BillListAPIResponse struct {
	BillList []json.RawMessage `json:"nzmimeepazxkubdpn"`
}

type BillListHead struct {
	Head []struct {
		ListTotalCount int `json:"list_total_count"`
		Result         struct {
			Code    string `json:"CODE"`
			Message string `json:"MESSAGE"`
		} `json:"RESULT"`
	} `json:"head"`
}

type BillListBody struct {
	Row []bill.BillRaw `json:"row"`
}

func FetchBillList(apiKey string, age string, page int, pageSize int) ([]bill.BillRaw, error) {
	base := "https://open.assembly.go.kr/portal/openapi/nzmimeepazxkubdpn"

	params := url.Values{}
	params.Add("KEY", apiKey)
	params.Add("Type", "json")
	params.Add("AGE", age)
	params.Add("pIndex", fmt.Sprint(page))
	params.Add("pSize", fmt.Sprint(pageSize))

	reqUrl := fmt.Sprintf("%s?%s", base, params.Encode())

	resp, err := util.MakeRequestWithUA("GET", reqUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var parsed BillListAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if len(parsed.BillList) < 2 {
		logging.Infof("[AGE=%s page=%d] no rows in response", age, page)
		return nil, ErrNoData
	}

	// 1. head
	var head BillListHead
	if err := json.Unmarshal(parsed.BillList[0], &head); err != nil {
		return nil, fmt.Errorf("failed to parse head: %v", err)
	}

	code := head.Head[1].Result.Code
	switch code {
	case "INFO-000":
		// 정상 처리
	case "INFO-200":
		return nil, ErrNoData
	default:
		return nil, fmt.Errorf("API returned error: CODE=%s MESSAGE=%s", code, head.Head[1].Result.Message)
	}

	// 2. row
	var bodyPart BillListBody
	if err := json.Unmarshal(parsed.BillList[1], &bodyPart); err != nil {
		return nil, fmt.Errorf("failed to parse rows: %v", err)
	}

	logging.Debugf("[AGE=%s page=%d] received %d bills", age, page, len(bodyPart.Row))
	return bodyPart.Row, nil
}

func FetchTotalBillCount(apiKey string, age string) (int, error) {
	base := "https://open.assembly.go.kr/portal/openapi/nzmimeepazxkubdpn"

	params := url.Values{}
	params.Add("KEY", apiKey)
	params.Add("Type", "json")
	params.Add("AGE", age)
	params.Add("pIndex", "1")
	params.Add("pSize", "1")

	reqUrl := fmt.Sprintf("%s?%s", base, params.Encode())

	resp, err := util.MakeRequestWithUA("GET", reqUrl)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %v", err)
	}

	var parsed BillListAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if len(parsed.BillList) < 1 {
		return 0, fmt.Errorf("unexpected API response")
	}

	var head BillListHead
	if err := json.Unmarshal(parsed.BillList[0], &head); err != nil {
		return 0, fmt.Errorf("failed to parse head: %v", err)
	}

	return head.Head[0].ListTotalCount, nil
}
