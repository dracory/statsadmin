package shared

import (
	"context"

	"github.com/dracory/statsstore"
)

// FlagIPVisitorsAsBot sets bot='yes' on all visitor records for the given IP
// in a single bulk UPDATE. Returns the number of visitor records that were
// found (regardless of whether they were already flagged). If no visitor
// records exist for the IP, returns (0, nil) — callers should check the
// count and warn the user accordingly.
func FlagIPVisitorsAsBot(ctx context.Context, store statsstore.StoreInterface, ip string) (int, error) {
	count, err := store.VisitorCount(ctx, statsstore.VisitorQuery().SetIPIn([]string{ip}))
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}

	if _, err := store.VisitorSetBotByIP(ctx, ip, statsstore.VALUE_YES); err != nil {
		return 0, err
	}
	return int(count), nil
}
