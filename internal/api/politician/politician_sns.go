package politician

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/model/politician"
)

type SNSAPIResponse struct {
	Negnlnyvatsjwocar []struct {
		Head []struct {
			ListTotalCount int `json:"list_total_count"`
			Result         struct {
				Code    string `json:"CODE"`
				Message string `json:"MESSAGE"`
			} `json:"RESULT"`
		} `json:"head"`
		Row []politician.PoliticianSNSRaw `json:"row"`
	} `json:"negnlnyvatsjwocar"`
}

// 국회의원 SNS API 호출
func FetchPoliticianSNS(apiKey string, page int, pageSize int) ([]politician.PoliticianSNSRaw, error) {
	url := fmt.Sprintf("https://open.assembly.go.kr/portal/openapi/negnlnyvatsjwocar?KEY=%s&Type=json&pIndex=%d&pSize=%d", apiKey, page, pageSize)

	resp, err := util.MakeRequestWithUA("GET", url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response: %v", err)
	}

	var parsed SNSAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %v", err)
	}

	if len(parsed.Negnlnyvatsjwocar) < 2 {
		return []politician.PoliticianSNSRaw{}, nil
	}

	return parsed.Negnlnyvatsjwocar[1].Row, nil
}
