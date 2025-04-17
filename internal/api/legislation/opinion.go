package legislation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"gwatch-data-pipeline/internal/logging"
	model "gwatch-data-pipeline/internal/model"
)

// 입법예고 의견 목록 Xlsx 다운로드하는 함수
func DownloadOpinionXlsxWithSession(session model.SessionInfo, billID string) error {
	logging.Infof("📥 [worker reuse] Downloading opinion Excel for bill_id: %s", billID)

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to get current directory: %v", err)
	}
	fileName := fmt.Sprintf("%s,%s.xlsx", billID, time.Now().Format("0601021504"))
	downloadPath := filepath.Join(wd, "downloads/opinion", fileName)
	os.MkdirAll(filepath.Dir(downloadPath), os.ModePerm)

	url := "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId=" + billID + "&searchConClosed=0"
	req, err := BuildOpinionDownloadRequest(session.CSRFToken, billID, session.Cookies, url)
	if err != nil {
		return fmt.Errorf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Server responded with status %d:\n%s", resp.StatusCode, string(body))
	}

	err = SaveResponseToFile(resp, downloadPath)
	if err != nil {
		return fmt.Errorf("Failed to save file: %v", err)
	}

	logging.Infof(" File downloaded to: %s", downloadPath)
	return nil
}

func WarmUpSessionWithViewPage(ctx context.Context, billID string) error {
	return chromedp.Run(ctx,
		chromedp.Navigate("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOngoing/view.do?lgsltPaId="+billID),
		chromedp.Navigate("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOngoing/list.do?lgsltPaId="+billID+"&menuNo=1100026"),
		chromedp.Navigate("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId="+billID+"&searchConClosed=0"),
		chromedp.WaitVisible(`#tbody_opnList`, chromedp.ByID),

		// 사용자가 셀렉트 박스 등 상호작용한 것처럼 이벤트 유도
		chromedp.Evaluate(`document.querySelector('select[name="pageUnit"]').value = "10";`, nil),
		chromedp.Evaluate(`document.querySelector('select[name="pageUnit"]').dispatchEvent(new Event("change"));`, nil),

		// iframe 로딩 유도
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`window.dispatchEvent(new Event("load"))`, nil),
		chromedp.Sleep(2*time.Second),
	)
}

// 의견 다운로드용 POST 요청을 구성하는 함수
func BuildOpinionDownloadRequest(csrfToken, billID string, cookies []*http.Cookie, url string) (*http.Request, error) {
	form := urlpkg.Values{
		"_csrf":           {csrfToken},
		"lgsltPaId":       {billID},
		"excelFileName":   {"입법예고 등록의견"},
		"headers":         {"의견번호,제목,작성자,의견제출기관,등록일"},
		"columns":         {"opnNo,sj,rgrNm,opnSbmInstNm,opnRgDt"},
		"menuNo":          {"1100026"},
		"sortCol":         {"OPN_NO"},
		"sortGbn":         {"DESC"},
		"searchConRng":    {"0"},
		"searchConKey":    {"0"},
		"searchWrd":       {""},
		"divType":         {""},
		"pageIndex":       {"1"},
		"pageUnit":        {"10"},
	}

	req, err := http.NewRequest("POST", "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/downloadExcel.uxls", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId=" + billID + "&searchConClosed=0")
	req.Header.Set("Origin", "https://pal.assembly.go.kr")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	req.AddCookie(&http.Cookie{Name: "fileDownloadToken", Value: "TRUE"})
	return req, nil
}
