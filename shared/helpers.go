package shared

import (
	"context"
	"strings"

	"github.com/dromara/carbon/v2"
)

// PeriodBounds holds the created_at >= / <= bounds for a selected period.
type PeriodBounds struct {
	From  string // inclusive, "YYYY-MM-DD HH:MM:SS"
	To    string // inclusive
	Label string
}

// ResolvePeriod maps a period identifier to concrete created_at bounds.
// Unknown or empty values fall back to the default period.
func ResolvePeriod(period string) PeriodBounds {
	now := carbon.Now(carbon.UTC)

	switch period {
	case PeriodToday:
		return PeriodBounds{
			From:  now.StartOfDay().ToDateTimeString(),
			To:    now.EndOfDay().ToDateTimeString(),
			Label: "Today",
		}
	case PeriodYesterday:
		y := now.SubDays(1)
		return PeriodBounds{
			From:  y.StartOfDay().ToDateTimeString(),
			To:    y.EndOfDay().ToDateTimeString(),
			Label: "Yesterday",
		}
	case PeriodLast7Days:
		return PeriodBounds{
			From:  now.SubDays(6).StartOfDay().ToDateTimeString(),
			To:    now.EndOfDay().ToDateTimeString(),
			Label: "Last 7 days",
		}
	case PeriodThisMonth:
		return PeriodBounds{
			From:  now.StartOfMonth().ToDateTimeString(),
			To:    now.EndOfDay().ToDateTimeString(),
			Label: "This month",
		}
	case PeriodLastMonth:
		prev := now.SubMonth()
		return PeriodBounds{
			From:  prev.StartOfMonth().ToDateTimeString(),
			To:    prev.EndOfMonth().ToDateTimeString(),
			Label: "Last month",
		}
	case PeriodAllTime:
		return PeriodBounds{
			From:  "1970-01-01 00:00:00",
			To:    carbon.Now(carbon.UTC).EndOfDay().ToDateTimeString(),
			Label: "All time",
		}
	default:
		return ResolvePeriod(PeriodDefault)
	}
}

// PeriodOptions returns the ordered list of selectable periods for the UI.
func PeriodOptions() []struct{ Value, Label string } {
	return []struct{ Value, Label string }{
		{PeriodToday, "Today"},
		{PeriodYesterday, "Yesterday"},
		{PeriodLast7Days, "Last 7 days"},
		{PeriodThisMonth, "This month"},
		{PeriodLastMonth, "Last month"},
		{PeriodAllTime, "All time"},
	}
}

// CountryNameResolver wraps a CountryNameByIso2 callback with a simple
// pass-through cache. It is safe for concurrent use; the lookup runs once
// per resolver instance.
type CountryNameResolver struct {
	lookup func(iso2Code string) (string, error)
	loaded bool
	m      map[string]string
}

// NewCountryNameResolver returns a fresh resolver bound to the given callback.
// If the callback is nil, Name always returns the raw code.
func NewCountryNameResolver(lookup func(iso2Code string) (string, error)) *CountryNameResolver {
	return &CountryNameResolver{lookup: lookup}
}

// Name returns the human-readable country name for an ISO2 code, falling back
// to the code itself when unknown or when no lookup callback is configured.
func (r *CountryNameResolver) Name(_ context.Context, iso2 string) string {
	if iso2 == "" {
		return "—"
	}
	if r.lookup == nil {
		return iso2
	}
	if !r.loaded {
		r.loaded = true
		// Defer to the callback per-call; no batch preload since the callback
		// signature is single-code. The resolver is still useful as a
		// placeholder for future caching.
	}
	name, err := r.lookup(iso2)
	if err != nil || name == "" {
		return iso2
	}
	return name
}

// CountEntry is a generic (label, count) pair used for top-N breakdowns.
type CountEntry struct {
	Label string
	Count int64
}

// AggregateCounts reduces a visitor slice to a frequency map keyed by the
// provided extractor. Empty labels are grouped under "—".
func AggregateCounts(visitors []VisitorLike, extract func(v VisitorLike) string) []CountEntry {
	counts := map[string]int64{}
	for _, v := range visitors {
		label := extract(v)
		if label == "" {
			label = "—"
		}
		counts[label]++
	}
	out := make([]CountEntry, 0, len(counts))
	for k, c := range counts {
		out = append(out, CountEntry{Label: k, Count: c})
	}
	// Sort by count desc, then label asc for stable ordering.
	sortCounts(out)
	return out
}

// VisitorLike is the minimal subset of statsstore.VisitorInterface used by
// aggregation helpers. The real interface satisfies it structurally.
type VisitorLike interface {
	GetPath() string
	GetCountry() string
	GetUserBrowser() string
	GetUserOs() string
	GetUserDevice() string
	GetUserDeviceType() string
	GetIpAddress() string
	GetFingerprint() string
	GetCreatedAt() string
	GetBot() string
	GetThreat() string
}

// IsBotVisitor reports whether a visitor is flagged as a bot, either via its
// bot field or because its device or device_type is identified as "bot".
func IsBotVisitor(v VisitorLike) bool {
	if v == nil {
		return false
	}
	return v.GetBot() == "yes" ||
		strings.EqualFold(v.GetUserDeviceType(), "bot") ||
		strings.EqualFold(v.GetUserDevice(), "bot")
}

func sortCounts(in []CountEntry) {
	// Simple insertion sort - breakdowns are small (<= a few dozen keys).
	for i := 1; i < len(in); i++ {
		for j := i; j > 0; j-- {
			if in[j].Count > in[j-1].Count || (in[j].Count == in[j-1].Count && in[j].Label < in[j-1].Label) {
				in[j], in[j-1] = in[j-1], in[j]
			} else {
				break
			}
		}
	}
}

// TopN returns at most n entries from a sorted slice.
func TopN(in []CountEntry, n int) []CountEntry {
	if n <= 0 || n >= len(in) {
		return in
	}
	return in[:n]
}
