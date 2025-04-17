package bill

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"gwatch-data-pipeline/internal/api/util"
	"gwatch-data-pipeline/internal/db"
	"gwatch-data-pipeline/internal/logging"
	"gwatch-data-pipeline/internal/model/bill"
)

// MEMBER_LIST URL로 이동해 발의자 명단을 파싱하고 DB에 매핑하는 함수
func FetchAndMatchProposers(billID string, memberListURL string, age string) ([]bill.BillPoliticianRelation, error) {
	resp, err := util.MakeRequestWithUA("GET", memberListURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch MEMBER_LIST page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MEMBER_LIST returned non-200: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read MEMBER_LIST response: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse MEMBER_LIST HTML: %v", err)
	}

	var relations []bill.BillPoliticianRelation

	doc.Find("div.layerInScroll a").Each(func(i int, s *goquery.Selection) {
		fullText := strings.TrimSpace(s.Text())
		name, hanja, party := parseProposerText(fullText)
		if name == "" {
			logging.Warnf("proposer missing name: %s", fullText)
			return
		}

		pid, err := filterPoliticianByNameAndParty(name, hanja, party, age)
		if err != nil {
			log.Println(err)
			return
		}

		role := "공동발의"
		if i == 0 {
			role = "대표발의"
		}

		relations = append(relations, bill.BillPoliticianRelation{
			BillID:       billID,
			PoliticianID: *pid,
			Role:         role,
		})
	})

	return relations, nil
}

func filterPoliticianByNameAndParty(name, hanja, party, age string) (*uint64, error) {
	billUnit, _ := strconv.Atoi(age)

	type Candidate struct {
		ID     uint64
		MonaCD string
		Unit   int
		Party  string
		Hanja  string
	}

	var candidates []Candidate
	err := db.DB.Table("politicians AS p").
		Select("p.id, p.mona_cd, p.hanja_name, t.unit, t.party").
		Joins("JOIN politician_terms AS t ON p.id = t.politician_id").
		Where("p.name = ?", name).
		Find(&candidates).Error
	if err != nil {
		return nil, fmt.Errorf("DB query failed: %v", err)
	}

	//  Step 1: mona_cd 유일
	monaSet := map[string]Candidate{}
	for _, c := range candidates {
		monaSet[c.MonaCD] = c
	}
	if len(monaSet) == 1 {
		for _, c := range monaSet {
			return &c.ID, nil
		}
	}

	//  Step 2: 한자명 기준
	var step2 []Candidate
	for _, c := range candidates {
		if c.Hanja == hanja {
			step2 = append(step2, c)
		}
	}
	if len(step2) == 1 {
		return &step2[0].ID, nil
	}

	//  Step 3: 정당 기준
	var step3 []Candidate
	for _, c := range candidates {
		if c.Party == party {
			step3 = append(step3, c)
		}
	}
	if len(step3) == 1 {
		return &step3[0].ID, nil
	}

	// Step 4: 정당 + 대수 기준
	var step4 []Candidate
	for _, c := range step3 {
		if c.Unit == billUnit {
			step4 = append(step4, c)
		}
	}
	if len(step4) == 1 {
		return &step4[0].ID, nil
	}

	// Step 5: 이름 + 대수 기준 fallback
	var step5 []Candidate
	for _, c := range candidates {
		if c.Unit == billUnit {
			step5 = append(step5, c)
		}
	}
	monaSet5 := map[string]uint64{}
	for _, c := range step5 {
		monaSet5[c.MonaCD] = c.ID
	}
	if len(monaSet5) == 1 {
		for _, id := range monaSet5 {
			log.Printf("⚠️ fallback match used (정당 무시): %s (%s / %d대)", name, party, billUnit)
			return &id, nil
		}
	}

	if len(step5) > 1 {
		for _, c := range step5 {
			log.Printf("🔍 ambiguous fallback candidate: mona_cd=%s, unit=%d, party=%s", c.MonaCD, c.Unit, c.Party)
		}
		return nil, fmt.Errorf("multiple candidates for %s (%s / %d대)", name, party, billUnit)
	}

	return nil, fmt.Errorf("no match found for %s (%s / %d대)", name, party, billUnit)
}

func parseProposerText(text string) (string, string, string) {
	text = strings.TrimSpace(text)

	// 케이스 1: (정당/한자) 형태
	if strings.Contains(text, "(") && strings.Contains(text, "/") && strings.Contains(text, ")") {
		parts := strings.SplitN(text, "(", 2)
		name := strings.TrimSpace(parts[0])
		rest := strings.TrimSuffix(parts[1], ")")
		subparts := strings.Split(rest, "/")
		if len(subparts) == 2 {
			party := strings.TrimSpace(subparts[0])
			hanja := strings.TrimSpace(subparts[1])
			return name, hanja, party
		}
	}

	// 케이스 2: 이름만 있는 경우 (ex. 홍성우)
	if !strings.ContainsAny(text, "()/") && len([]rune(text)) >= 2 {
		return text, "", ""
	}

	// 파싱 실패
	return "", "", ""
}