package utils

import (
	"context"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func SetAllocCookie(cookie *http.Cookie) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var expr cdp.TimeSinceEpoch
		if cookie.Expires.IsZero() {
			expr = cdp.TimeSinceEpoch(time.Now().Add(365 * 24 * time.Hour))
		} else {
			expr = cdp.TimeSinceEpoch(cookie.Expires)
		}

		err := network.SetCookie(cookie.Name, cookie.Value).
			WithExpires(&expr).
			WithDomain(cookie.Domain).
			WithPath(cookie.Path).
			WithHTTPOnly(cookie.HttpOnly).
			WithSecure(cookie.Secure).
			Do(ctx)
		if err != nil {
			return err
		}
		return nil
	})
}

func SetAllocCookies(cookies []*http.Cookie) []chromedp.Action {
	cookiesActions := make([]chromedp.Action, 0, len(cookies))
	for _, cookie := range cookies {
		cookiesActions = append(
			cookiesActions,
			SetAllocCookie(cookie),
		)
	}
	return cookiesActions
}

func GetDefaultChromedpAlloc(userAgent string) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(userAgent),
	)
	return chromedp.NewExecAllocator(context.Background(), opts...)
}

func ExecuteChromedpActions(allocCtx context.Context, allocCancelFn context.CancelFunc, actions ...chromedp.Action) error {
	if allocCtx == nil {
		allocCtx = context.Background()
	}

	taskCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	return chromedp.Run(taskCtx, actions...)
}
