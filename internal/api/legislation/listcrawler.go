package legislation

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"

	"gwatch-data-pipeline/internal/logging"
	model "gwatch-data-pipeline/internal/model/legislation"
)

// bill_no로 bill_id를 찾아 반환하는 함수
func GetBillIDByNo(billNo string, db *gorm.DB) (string, error) {
    var billID string
    if err := db.Model(&model.Bill{}).Where("bill_no = ?", billNo).Select("bill_id").Scan(&billID).Error; err != nil {
        logging.Errorf("Failed to fetch bill ID: %v", err)
        return "", err
    }
    return billID, nil
}

// URL로부터 입법예고기간과 의견 수를 가져오는 함수
func FetchNoticePeriodFast(url string) (string, int, error) {
    client := &http.Client{}
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        logging.Errorf("Failed to create HTTP request: %v", err)
        return "", 0, err
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

    res, err := client.Do(req)
    if err != nil {
        logging.Errorf("HTTP request failed: %v", err)
        return "", 0, err
    }
    defer res.Body.Close()

    doc, err := goquery.NewDocumentFromReader(res.Body)
    if err != nil {
        logging.Errorf("Failed to parse HTML: %v", err)
        return "", 0, err
    }

    var period string
    doc.Find("ul.m_date li").EachWithBreak(func(i int, s *goquery.Selection) bool {
        text := strings.TrimSpace(s.Text())
        if strings.HasPrefix(text, "입법예고기간 :") {
            period = strings.TrimPrefix(text, "입법예고기간 :")
            return false
        }
        return true
    })

    commentCountStr := doc.Find("div.board_count strong").First().Text()
    commentCount, _ := strconv.Atoi(strings.TrimSpace(commentCountStr))

    if period == "" {
        logging.Errorf("입법예고기간 not found in HTML at: %s",url)
        return "", 0, fmt.Errorf("notice period not found")
    }

    return period, commentCount, nil
}
