package ipdetails

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
	"github.com/dracory/req"
	"github.com/dracory/statsstore"
)

func (controller *ipDetailsController) handleLoadPaths(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	type condition struct {
		Field    string `json:"field"`
		Operator string `json:"operator"`
		Value    string `json:"value"`
	}

	var body struct {
		IP         string      `json:"ip"`
		Conditions []condition `json:"conditions"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	ip := req.GetStringTrimmed(r, shared.ParamIP)
	if ip == "" {
		ip = strings.TrimSpace(body.IP)
	}
	if ip == "" {
		api.Respond(w, r, api.Error("IP address is required"))
		return ""
	}

	excludeBots := false
	for _, c := range body.Conditions {
		if c.Field == "is_bot" && strings.EqualFold(strings.TrimSpace(c.Value), "no") {
			excludeBots = true
			break
		}
	}

	ctx := r.Context()

	// Fetch all visitors for this IP. The store query doesn't support
	// filtering by IP directly, so we fetch a large batch and filter
	// in-memory. PathContains with empty string matches all paths.
	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetOrderBy("created_at").
		SetSortOrder("DESC").
		SetLimit(10000))
	if err != nil {
		slog.Error("ipdetails: failed to load visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to load visitor data"))
		return ""
	}

	// Filter to just this IP (and optionally exclude bot visits if is_bot=no condition is set).
	var ipVisitors []statsstore.VisitorInterface
	for _, v := range visitors {
		if v.GetIpAddress() == ip {
			if excludeBots && shared.IsBotVisitor(v) {
				continue
			}
			ipVisitors = append(ipVisitors, v)
		}
	}

	if len(ipVisitors) == 0 {
		api.Respond(w, r, api.SuccessWithData("No visits found", map[string]any{
			FieldDetails:    map[string]any{},
			FieldPaths:      []any{},
			FieldVisitCount: 0,
			FieldIsBot:      false,
			FieldIsThreat:   false,
		}))
		return ""
	}

	// Build details from the most recent record (DESC sort).
	latest := ipVisitors[0]
	first := ipVisitors[len(ipVisitors)-1]

	// Build path history (already DESC by created_at).
	paths := make([]map[string]any, 0, len(ipVisitors))
	for _, v := range ipVisitors {
		paths = append(paths, map[string]any{
			FieldPath:          v.GetPath(),
			FieldPathCreatedAt: v.GetCreatedAt(),
		})
	}

	// Unique path count.
	uniquePaths := map[string]bool{}
	for _, v := range ipVisitors {
		uniquePaths[v.GetPath()] = true
	}

	details := map[string]any{
		FieldIP:          ip,
		FieldCountry:     latest.GetCountry(),
		FieldCountryName: controller.opts.CountryName(latest.GetCountry()),
		FieldBrowser:     latest.GetUserBrowser(),
		FieldBrowserVer:  latest.GetUserBrowserVersion(),
		FieldOS:          latest.GetUserOs(),
		FieldOSVer:       latest.GetUserOsVersion(),
		FieldDevice:      latest.GetUserDevice(),
		FieldDeviceType:  latest.GetUserDeviceType(),
		FieldUserAgent:   latest.GetUserAgent(),
		FieldAcceptLang:  latest.GetUserAcceptLanguage(),
		FieldAcceptEnc:   latest.GetUserAcceptEncoding(),
		FieldReferrer:    latest.GetUserReferrer(),
		FieldFingerprint: latest.GetFingerprint(),
		FieldFirstSeen:   first.GetCreatedAt(),
		FieldLastSeen:    latest.GetCreatedAt(),
	}

	// Sort paths by created_at descending (already sorted, but ensure).
	sort.Slice(paths, func(i, j int) bool {
		ci, _ := paths[i][FieldPathCreatedAt].(string)
		cj, _ := paths[j][FieldPathCreatedAt].(string)
		return ci > cj
	})

	// Compute bot reasons from the visitor records so the UI can show why
	// this IP was flagged. Uses the shared helper that checks all ingestion
	// signals (user-agent, data-center IP, referrer spam, bot/malicious paths).
	botReasons := shared.ComputeBotReasons(ipVisitors, nil)

	api.Respond(w, r, api.SuccessWithData("IP details loaded", map[string]any{
		FieldDetails:    details,
		FieldPaths:      paths,
		FieldVisitCount: len(ipVisitors),
		FieldIsBot:      shared.IsBotVisitor(latest),
		FieldIsThreat:   latest.GetThreat() == statsstore.VALUE_YES,
		FieldBotReasons: botReasons,
		FieldTotal:      len(paths),
	}))
	return ""
}
