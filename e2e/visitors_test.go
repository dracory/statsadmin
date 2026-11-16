package e2e

import (
	"testing"
)

// TestVisitors verifies the visitors page mounts and renders the seeded
// visitor rows in the results table.
func TestVisitors(t *testing.T) {
	// Wait for the second body row: data rows are produced by v-for, so a
	// second row only exists after Vue mounted and loaded the list.
	body := waitSelector(t, newTab(t), adminURL("visitors"),
		"#stats-visitors-app table tbody tr:nth-of-type(2)")

	assertContains(t, body, seedIP, seedIP2, seedPath)
}
