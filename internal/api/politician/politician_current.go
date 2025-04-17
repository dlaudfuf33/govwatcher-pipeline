package politician

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/model/politician"
)

type CurrentAPIResponse struct {
	Nwvrqwxyaytdsfvhu []struct {
		Head []struct {
			ListTotalCount int `json:"list_total_count"`
			Result         struct {
				Code    string `json:"CODE"`
				Message string `json:"MESSAGE"`
			} `json:"RESULT"`
		} `json:"head"`
		Row []politician.PoliticianRaw `json:"row"`
	} `json:"nwvrqwxyaytdsfvhu"`
}
// 현역 국회의원 인적사항 API
func FetchCurrentPoliticians(apiKey string, page int, pageSize int) ([]politician.PoliticianRaw, error) {
	url := fmt.Sprintf("https://open.assembly.go.kr/portal/openapi/nwvrqwxyaytdsfvhu?KEY=%s&Type=json&pIndex=%d&pSize=%d", apiKey, page, pageSize)

	resp, err := util.MakeRequestWithUA("GET", url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed read response : %v", err)
	}

	var parsed CurrentAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed parsing JSON : %v", err)
	}

	if len(parsed.Nwvrqwxyaytdsfvhu) < 2 {
		return []politician.PoliticianRaw{}, nil
	}

	return parsed.Nwvrqwxyaytdsfvhu[1].Row, nil
}
