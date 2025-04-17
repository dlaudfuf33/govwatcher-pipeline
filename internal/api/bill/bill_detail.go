package bill

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"gwatch-data-pipeline/internal/api/util"
)

// FetchBillDetailInfo: 상세페이지에서 제안이유 + 진행단계 파싱
func FetchBillDetailInfo(detailURL string) (string, string, string, error) {
	
	resp, err := util.MakeRequestWithUA("GET", detailURL)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to GET detail page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf(" non-200 status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf(" failed to read body: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", "", "", fmt.Errorf(" failed to parse HTML: %v", err)
	}

	// 제안이유 및 주요내용
	summary := strings.TrimSpace(doc.Find("#summaryContentDiv").Text())
	summary = cleanText(summary)

	// 전체 단계 로그
	stepSel := doc.Find("div.stepType01 span")
	var stepLogParts []string
	currentStep := ""
	stepSel.Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			stepLogParts = append(stepLogParts, text)
			if class, exists := s.Attr("class"); exists && strings.Contains(class, "on") {
				currentStep = text
			}
		}
	})
	stepLog := strings.Join(stepLogParts, " > ")

	return summary, stepLog, currentStep, nil
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.ReplaceAll(s, "\t", "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	return s
}
