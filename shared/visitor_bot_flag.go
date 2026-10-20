package shared

import (
	"context"

	"github.com/dracory/statsstore"
)

// FlagIPVisitorsAsBot sets bot='yes' on all visitor records for the given IP.
// Returns the number of visitor records that were found (regardless of whether
// they were already flagged). If no visitor records exist for the IP, returns
// (0, nil) — callers should check the count and warn the user accordingly.
func FlagIPVisitorsAsBot(ctx context.Context, store statsstore.StoreInterface, ip string) (int, error) {
	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetIPIn([]string{ip}).
		SetLimit(100000))
	if err != nil {
		return 0, err
	}

	count := 0
	for _, v := range visitors {
		count++
		if v.GetBot() == statsstore.VALUE_YES {
			continue
		}
		v.SetBot(statsstore.VALUE_YES)
		if err := store.VisitorUpdate(ctx, v); err != nil {
			return count, err
		}
	}
	return count, nil
}
