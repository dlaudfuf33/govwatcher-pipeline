package politician

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/logging"
	"gwatch-data-pipeline/internal/model/politician"
)
type HistoryAPIResponse struct {
	Npffdutiapkzbfyvr []json.RawMessage `json:"npffdutiapkzbfyvr"`
}

type HeadWrapper struct {
	Head []struct {
		ListTotalCount int `json:"list_total_count"`
		Result         struct {
			Code    string `json:"CODE"`
			Message string `json:"MESSAGE"`
		} `json:"RESULT"`
	} `json:"head"`
}

type RowWrapper struct {
	Row []politician.PoliticianRaw `json:"row"`
}
// 국회의원 이력 API 호출
func FetchHistoricalPoliticians(apiKey string, unitCd string, page int, pageSize int) ([]politician.PoliticianRaw, error) {
	url := fmt.Sprintf("https://open.assembly.go.kr/portal/openapi/npffdutiapkzbfyvr?KEY=%s&Type=json&pIndex=%d&pSize=%d&UNIT_CD=%s", apiKey, page, pageSize, unitCd)

	resp, err := util.MakeRequestWithUA("GET", url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(" Failed to read response body: %v", err)
	}
	
	var parsed HistoryAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf(" Failed to parse JSON: %v", err)
	}
	
	if len(parsed.Npffdutiapkzbfyvr) < 2 {
		logging.Infof("[unit_cd=%s page=%d] Response OK but no data (row missing)\n", unitCd, page)
		return nil, util.ErrNoData
	}
	
	var head HeadWrapper
	if err := json.Unmarshal(parsed.Npffdutiapkzbfyvr[0], &head); err != nil {
		return nil, fmt.Errorf(" Failed to parse head section: %v", err)
	}
	code := head.Head[1].Result.Code
	
	switch code {
	case "INFO-000":
	case "INFO-200":
		return nil, util.ErrNoData
	default:
		return nil, fmt.Errorf(" API error: CODE=%s MESSAGE=%s", code, head.Head[1].Result.Message)
	}
	
	var row RowWrapper
	if err := json.Unmarshal(parsed.Npffdutiapkzbfyvr[1], &row); err != nil {
		return nil, fmt.Errorf(" Failed to parse row section: %v", err)
	}
	
	logging.Debugf("📦 [unit_cd=%s page=%d] Number of rows: %d", unitCd, page, len(row.Row))
	
	return row.Row, nil
}	