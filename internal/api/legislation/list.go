package legislation

import (
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gwatch-data-pipeline/internal/logging"
)

// 진행 중 입법예고 Xlsx 다운로드하는 함수
func DownloadLegislativeListXlsx() error {
	// 세션 생성 및 쿠키 가져오기
	ctx, cancel := CreateChromedpContext()
	defer cancel()

	// 대상 페이지로 이동
	url := "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOngoing/list.do?searchConClosed=0&menuNo=1100026"
	err := warmUpSessionWithViewPage(ctx)
	if err != nil {
		logging.Errorf("Failed to warm up session: %v", err)
		return err
	}
	csrfToken, err := FetchCSRFToken(ctx, url)
	if err != nil {
		logging.Errorf("Failed to fetch CSRF token: %v", err)
		return fmt.Errorf("failed to fetch CSRF token: %w", err)
	}

	logging.Debugf("✅ CSRF token retrieved: %s", csrfToken)

	cookies, err := GetCookiesForRequest(ctx)
	if err != nil {
		logging.Errorf("Failed to get cookies: %v", err)
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// 파일 경로 설정
	wd, err := os.Getwd()
	if err != nil {
		logging.Errorf("Failed to get current directory: %v", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	fileName := fmt.Sprintf("legislation_notice_%s.xlsx", time.Now().Format("0601021504"))
	downloadPath := filepath.Join(wd, "downloads/notice", fileName)
	os.MkdirAll(filepath.Dir(downloadPath), os.ModePerm)

	// CSRF 토큰 디버깅 출력
	logging.Debugf("CSRF token: %s", csrfToken)

	// 쿠키 디버깅 출력
	logging.Debugf("Cookies being sent:")
	for _, c := range cookies {
		logging.Debugf("- %s = %s", c.Name, c.Value)
	}

	// 다운로드 요청 생성
	req, err := buildDownloadRequest(csrfToken, cookies, url)
	if err != nil {
		logging.Errorf("Failed to create request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 요청 실행
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logging.Errorf("Request failed: %v", err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// HTTP 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Errorf("Server responded with status %d:\n%s", resp.StatusCode, string(body))
		return fmt.Errorf("server responded with status %d: %s", resp.StatusCode, string(body))
	}

	// 파일 저장
	err = SaveResponseToFile(resp, downloadPath)
	if err != nil {
		logging.Errorf("Failed to save file: %v", err)
		return fmt.Errorf("failed to save file: %w", err)
	}

	logging.Infof("File downloaded to: %s", downloadPath)
	return nil
}

// 요청용 폼 데이터를 포함한 HTTP POST 요청을 생성하는 함수
func buildDownloadRequest(csrfToken string, cookies []*http.Cookie, url string) (*http.Request, error) {
	form := urlpkg.Values{
		"_csrf":           {csrfToken},
		"excelFileName":   {"진행 중 입법예고"},
		"headers":         {"의안번호,의견수,법률안명,제안자구분,소관위원회,주요내용,등록일시"},
		"columns":         {"billNo,opnCnt,billNameTitle,proposerKindCd,currCommittee,ppslRsonMnCn,lgsltPaRgDt"},
		"menuNo":          {"1100026"},
		"sortCol":         {"BILL_NO"},
		"sortGbn":         {"DESC"},
		"searchConClosed": {"0"},
		"pageIndex":       {"1"},
		"divType":         {""},
		"committeeId":     {""},
		"billName":        {""},
		"represent":       {""},
		"proposers":       {""},
		"ppslRsonMnCn":    {""},
		"committeeIdMb":   {""},
		"billNameMb":      {""},
		"representMb":     {""},
		"proposersMb":     {""},
		"ppslRsonMnCnMb":  {""},
		"pageUnit":        {"10"},
	}

	req, err := http.NewRequest("POST", "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOngoing/downloadExcel.uxls", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", url)
	req.Header.Set("Origin", "https://pal.assembly.go.kr")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req, nil
}
