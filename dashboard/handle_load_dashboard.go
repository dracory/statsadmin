package dashboard

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

type condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func applyStoreConditions(query statsstore.VisitorQueryInterface, conds []condition) {
	for _, c := range conds {
		val := strings.TrimSpace(c.Value)
		if val == "" {
			continue
		}
		switch c.Field {
		case "country":
			query.SetCountry(val)
		case "device_type":
			query.SetDeviceType(val)
		case "path_contains":
			query.SetPathContains(val)
		case "path_exact":
			query.SetPathExact(val)
		case "date_from":
			query.SetCreatedAtGte(val + " 00:00:00")
		case "date_to":
			query.SetCreatedAtLte(val + " 23:59:59")
		case "is_bot":
			if strings.EqualFold(val, "yes") {
				query.SetBot(statsstore.VALUE_YES)
			} else if strings.EqualFold(val, "no") {
				query.SetBot(statsstore.VALUE_NO)
			}
		case "is_threat":
			if strings.EqualFold(val, "yes") {
				query.SetThreat(statsstore.VALUE_YES)
			} else if strings.EqualFold(val, "no") {
				query.SetThreat(statsstore.VALUE_NO)
			}
		}
	}
}

func (controller *dashboardController) handleLoadDashboard(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	var reqBody struct {
		Period     string      `json:"period"`
		Conditions []condition `json:"conditions"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	if reqBody.Period == "" {
		reqBody.Period = shared.PeriodDefault
	}

	bounds := shared.ResolvePeriod(reqBody.Period)
	ctx := r.Context()

	query := statsstore.VisitorQuery().
		SetCreatedAtGte(bounds.From).
		SetCreatedAtLte(bounds.To).
		SetLimit(5000)

	applyStoreConditions(query, reqBody.Conditions)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin dashboard: failed to load visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to load visitor data: "+err.Error()))
		return ""
	}

	// Adapt statsstore.VisitorInterface to shared.VisitorLike for aggregation.
	like := make([]shared.VisitorLike, 0, len(visitors))
	for _, v := range visitors {
		like = append(like, visitorAdapter{v: v})
	}

	totalVisitors := int64(len(like))
	uniqueVisitors := countUniqueVisitors(like)

	topPaths := shared.TopN(shared.AggregateCounts(like, func(v shared.VisitorLike) string { return v.GetPath() }), 10)
	topCountries := shared.TopN(shared.AggregateCounts(like, func(v shared.VisitorLike) string { return v.GetCountry() }), 10)
	topBrowsers := shared.TopN(shared.AggregateCounts(like, func(v shared.VisitorLike) string { return v.GetUserBrowser() }), 10)
	topOS := shared.TopN(shared.AggregateCounts(like, func(v shared.VisitorLike) string { return v.GetUserOs() }), 10)
	topDeviceTypes := shared.TopN(shared.AggregateCounts(like, func(v shared.VisitorLike) string { return v.GetUserDeviceType() }), 10)

	// Recent visitors (last 15, newest first — visitors are typically ordered by created_at desc).
	recent := buildRecentVisitors(like, 15, controller.opts)

	api.Respond(w, r, api.SuccessWithData("Dashboard loaded", map[string]any{
		FieldTotalVisitors:  totalVisitors,
		FieldUniqueVisitors: uniqueVisitors,
		FieldPeriod:         reqBody.Period,
		FieldPeriodLabel:    bounds.Label,
		FieldTopPaths:       toCountList(topPaths),
		FieldTopCountries:   toCountryCountList(topCountries, controller.opts),
		FieldTopBrowsers:    toCountList(topBrowsers),
		FieldTopOS:          toCountList(topOS),
		FieldTopDeviceTypes: toCountList(topDeviceTypes),
		FieldRecentVisitors: recent,
	}))
	return ""
}

// visitorAdapter wraps statsstore.VisitorInterface to satisfy shared.VisitorLike.
type visitorAdapter struct {
	v statsstore.VisitorInterface
}

func (a visitorAdapter) GetPath() string           { return a.v.GetPath() }
func (a visitorAdapter) GetCountry() string        { return a.v.GetCountry() }
func (a visitorAdapter) GetUserBrowser() string    { return a.v.GetUserBrowser() }
func (a visitorAdapter) GetUserOs() string         { return a.v.GetUserOs() }
func (a visitorAdapter) GetUserDevice() string     { return a.v.GetUserDevice() }
func (a visitorAdapter) GetUserDeviceType() string { return a.v.GetUserDeviceType() }
func (a visitorAdapter) GetIpAddress() string      { return a.v.GetIpAddress() }
func (a visitorAdapter) GetFingerprint() string    { return a.v.GetFingerprint() }
func (a visitorAdapter) GetCreatedAt() string      { return a.v.GetCreatedAt() }
func (a visitorAdapter) GetBot() string            { return a.v.GetBot() }
func (a visitorAdapter) GetThreat() string         { return a.v.GetThreat() }

func countUniqueVisitors(visitors []shared.VisitorLike) int64 {
	seen := map[string]struct{}{}
	for _, v := range visitors {
		if fp := v.GetFingerprint(); fp != "" {
			seen[fp] = struct{}{}
		}
	}
	return int64(len(seen))
}

func toCountList(in []shared.CountEntry) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, e := range in {
		out = append(out, map[string]any{
			FieldLabel: e.Label,
			FieldCount: e.Count,
		})
	}
	return out
}

// toCountryCountList is like toCountList but also resolves the ISO2 label to
// a human-readable country name via the CountryNameByIso2 callback.
func toCountryCountList(in []shared.CountEntry, opts shared.ControllerOptions) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, e := range in {
		out = append(out, map[string]any{
			FieldLabel:         e.Label,
			FieldLabelResolved: opts.CountryName(e.Label),
			FieldCount:         e.Count,
		})
	}
	return out
}

func buildRecentVisitors(visitors []shared.VisitorLike, n int, opts shared.ControllerOptions) []map[string]any {
	if n > len(visitors) {
		n = len(visitors)
	}
	out := make([]map[string]any, 0, n)
	for i := 0; i < n; i++ {
		v := visitors[i]
		out = append(out, map[string]any{
			FieldVisitorIP:          v.GetIpAddress(),
			FieldVisitorCountry:     v.GetCountry(),
			FieldVisitorCountryName: opts.CountryName(v.GetCountry()),
			FieldVisitorPath:        v.GetPath(),
			FieldVisitorBrowser:     v.GetUserBrowser(),
			FieldVisitorOS:          v.GetUserOs(),
			FieldVisitorDevice:      v.GetUserDeviceType(),
			FieldVisitorCreatedAt:   v.GetCreatedAt(),
			FieldIsBot:              shared.IsBotVisitor(v),
			FieldIsThreat:           v.GetThreat() == statsstore.VALUE_YES,
		})
	}
	return out
}
