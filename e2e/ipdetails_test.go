package e2e

import (
	"testing"
)

// TestIPDetails verifies the IP details page loads the seeded IP's
// summary card and path history.
func TestIPDetails(t *testing.T) {
	// The summary card only renders once the load-paths action returns
	// (the app shows a spinner while loading).
	body := waitSelector(t, newTab(t), adminURL("ip-details")+"&ip="+seedIP,
		"#ip-details-app .card")

	assertContains(t, body, seedIP, seedPath)
}
