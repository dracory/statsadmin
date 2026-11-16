package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/chromedp/chromedp"
)

// adminURL builds a controller URL, e.g. /admin/stats?controller=visitors.
func adminURL(controller string) string {
	return baseURL + adminPath + "?controller=" + controller
}

// waitSelector navigates to url, waits for sel to be ready, and returns
// the body's text content.
func waitSelector(t *testing.T, ctx context.Context, url, sel string) string {
	t.Helper()
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady(sel, chromedp.ByQuery),
		chromedp.Text("body", &body),
	)
	if err != nil {
		t.Fatalf("page %s failed: %v", url, err)
	}
	return body
}

// assertContains fails the test when body does not contain any of the
// expected substrings.
func assertContains(t *testing.T, body string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Fatalf("expected page to contain %q", want)
		}
	}
}
