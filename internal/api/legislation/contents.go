package legislation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"strings"
	"time"

	model "gwatch-data-pipeline/internal/model"
)

// 의견 본문 요청 API
func FetchOpinionContent(billID string, opnNo string, session model.SessionInfo) (string, time.Time, error) {
    form := urlpkg.Values{
        "lgsltPaId": {billID},
        "opnNo":     {opnNo},
    }

    req, err := http.NewRequest("POST", "https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/findOneLgsltpaOpnById.json", strings.NewReader(form.Encode()))
    if err != nil {
        return "",time.Time{},err
    }

    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("x-csrf-token", session.CSRFToken)
    req.Header.Set("Referer", fmt.Sprintf("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOpn/list.do?lgsltPaId=%s&searchConClosed=0", billID))
    req.Header.Set("Origin", "https://pal.assembly.go.kr")
    req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
    req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
    req.Header.Set("X-Requested-With", "XMLHttpRequest")
    req.Header.Set("requestAJAX", "true")

    for _, c := range session.Cookies {
        req.AddCookie(&http.Cookie{
            Name:  c.Name,
            Value: c.Value,
        })
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "",time.Time{},err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "",time.Time{},fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
    }

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "",time.Time{},err
    }
    
    var result struct {
        Result struct {
            Cn string `json:"cn"`
            OpnRgDt  string `json:"opnRgDt"`
        } `json:"result"`
    }
    fmt.Println("📦 response :", string(bodyBytes))

    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return "",time.Time{}, err
    }
    layout := "2006-01-02"
    parsedTime, err := time.Parse(layout, result.Result.OpnRgDt)
    if err != nil {
        return "", time.Time{}, fmt.Errorf("opnRgDt parsing error: %v", err)
    }
    return result.Result.Cn, parsedTime, nil
}