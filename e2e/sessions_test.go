package e2e

import (
	"testing"
)

// TestSessions verifies the sessions page mounts and groups the seeded
// visits into per-IP session cards.
func TestSessions(t *testing.T) {
	// Session cards only render after Vue mounted and loaded sessions;
	// each card's visits table contains the visit paths.
	body := waitSelector(t, newTab(t), adminURL("sessions"),
		"#stats-sessions-app .card table tbody tr")

	assertContains(t, body, seedIP, seedIP2, "visits")
}
