package legislation

import (
	"context"
	"net/http"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"gwatch-data-pipeline/internal/logging"
)

// chromedp 실행 컨텍스트를 생성하는 함수
func CreateChromedpContext() (context.Context, context.CancelFunc) {
	logging.Debugf("CreateChromedpContext called!")

    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", true),
        chromedp.Flag("blink-settings", "imagesEnabled=false"),
        chromedp.Flag("disable-background-networking", true),
        chromedp.Flag("disable-default-apps", true),
        chromedp.Flag("disable-extensions", true),
        chromedp.Flag("disable-sync", true),
        chromedp.Flag("disable-translate", true),
        chromedp.Flag("disable-gpu", true),
        chromedp.Flag("mute-audio", true),
    )
    allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
    ctx, cancelCtx := chromedp.NewContext(allocCtx)
    return ctx, func() {
        cancelCtx()
        cancel()
    }
}

// 현재 chromedp 세션에서 쿠키를 추출하는 함수
func GetCookiesForRequest(ctx context.Context) ([]*http.Cookie, error) {
	logging.Debugf("GetCookiesForRequest called!")
    
    var cookieParams []*network.Cookie
    err := chromedp.Run(ctx,
        network.Enable(),
        chromedp.ActionFunc(func(ctx context.Context) error {
            var err error
            cookieParams, err = network.GetCookies().Do(ctx)
            return err
        }),
    )
    if err != nil {
        logging.Errorf(" Failed to retrieve cookies: %v", err)
        return nil, err
    }
    logging.Debugf("Successfully retrieved cookies")

    var cookies []*http.Cookie
    for _, c := range cookieParams {
        cookies = append(cookies, &http.Cookie{
            Name:   c.Name,
            Value:  c.Value,
            Domain: c.Domain,
            Path:   c.Path,
        })
    }
    return cookies, nil
}

// 대상 페이지에서 CSRF 토큰을 추출하는 함수
func FetchCSRFToken(ctx context.Context, url string) (string, error) {
	logging.Debugf("FetchCSRFToken called!")
    
    var csrfToken string
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.WaitReady(`input[name="_csrf"]`),
        chromedp.Evaluate(`document.querySelector('input[name="_csrf"]').value`, &csrfToken),
    )
    if err != nil {
        logging.Errorf(" Failed to retrieve CSRF token: %v", err)
        return "", err
    }
    logging.Debugf("CSRF token retrieved: %s", csrfToken)
    return csrfToken, nil
}

// view.do 페이지에 접속해 세션을 준비하는 함수
func warmUpSessionWithViewPage(ctx context.Context) error {
    return chromedp.Run(ctx,
        chromedp.Navigate("https://pal.assembly.go.kr/napal/lgsltpa/lgsltpaOngoing/view.do?lgsltPaId=placeholder"),
        chromedp.WaitReady("body"),
    )
}
