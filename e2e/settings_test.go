package e2e

import (
	"testing"

	"github.com/chromedp/chromedp"
)

// TestSettings verifies the settings page mounts (v-cloak removed) and
// renders the bot management section.
func TestSettings(t *testing.T) {
	ctx := newTab(t)

	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(adminURL("settings")),
		// Vue removes v-cloak once the app mounts — this selector only
		// matches after mount.
		chromedp.WaitReady("#stats-settings-app:not([v-cloak])", chromedp.ByQuery),
		chromedp.Text("body", &body),
	)
	if err != nil {
		t.Fatalf("settings page failed: %v", err)
	}

	assertContains(t, body, "Settings", "Bot")
}
