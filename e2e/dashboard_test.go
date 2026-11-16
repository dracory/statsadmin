package e2e

import (
	"testing"
)

// TestDashboard verifies the dashboard mounts and renders the stat cards
// and the Top Paths table populated from seeded visitors.
func TestDashboard(t *testing.T) {
	// The Top Paths table rows only exist after Vue mounted and the
	// load-dashboard action returned data.
	body := waitSelector(t, newTab(t), adminURL("dashboard"),
		"#stats-dashboard-app table tbody tr")

	// innerText reflects the rendered text — card subtitles are uppercased
	// by the text-uppercase CSS class.
	assertContains(t, body,
		"TOTAL VISITORS",
		"UNIQUE VISITORS",
		"Top Paths",
		seedPath,
	)
}
