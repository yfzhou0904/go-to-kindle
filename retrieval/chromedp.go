package retrieval

import (
	"context"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type ChromedpMethod struct {
	execPath string
}

func NewChromedpMethod(execPath string) *ChromedpMethod {
	return &ChromedpMethod{execPath: execPath}
}

func (c *ChromedpMethod) Name() string {
	return "Chromedp"
}

func (c *ChromedpMethod) CanHandle(u *url.URL) bool {
	return strings.HasPrefix(u.String(), "http://") || strings.HasPrefix(u.String(), "https://")
}

func (c *ChromedpMethod) Retrieve(u *url.URL) *Result {
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"),
	)
	if c.execPath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(c.execPath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(u.String()),
		waitDocumentReady(),
		chromedp.Sleep(1500*time.Millisecond),
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	)
	if err != nil {
		return &Result{Error: err}
	}

	return &Result{
		Content: io.NopCloser(strings.NewReader(html)),
		URL:     u,
	}
}

func waitDocumentReady() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		for {
			var ready string
			if err := chromedp.Evaluate(`document.readyState`, &ready).Do(ctx); err != nil {
				return err
			}
			if ready == "complete" {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}
