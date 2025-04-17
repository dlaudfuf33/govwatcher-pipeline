package politician

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/model/politician"
)

// AllAPIResponse는 통합 국회의원 정보 API 응답 형식
type AllAPIResponse struct {
	Allnamember []struct {
		Head []struct {
			ListTotalCount int `json:"list_total_count"`
			Result         struct {
				Code    string `json:"CODE"`
				Message string `json:"MESSAGE"`
			} `json:"RESULT"`
		} `json:"head"`
		Row []politician.PoliticianRaw `json:"row"`
	} `json:"allnamember"`
}
// 역대 국회의원 인적사항 API 호출
func FetchAllPoliticians(apiKey string, page int, pageSize int) ([]politician.PoliticianRaw, error) {
	url := fmt.Sprintf("https://open.assembly.go.kr/portal/openapi/ALLNAMEMBER?KEY=%s&Type=json&pIndex=%d&pSize=%d", apiKey, page, pageSize)

	resp, err := util.MakeRequestWithUA("GET", url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %v", err)
	}

	var parsed AllAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %v", err)
	}

	if len(parsed.Allnamember) < 2 {
		return []politician.PoliticianRaw{}, nil
	}

	return parsed.Allnamember[1].Row, nil
}
