package settings

import (
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/dracory/api"
	"github.com/dracory/statsadmin/shared"
	"github.com/dracory/statsstore"
)

// reasonAgg tracks per-pattern evidence for a bot IP during aggregation.
type reasonAgg struct {
	Pattern string
	Hits    int
	Paths   []string
}

func (controller *settingsController) handleLoadBots(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	visitors, err := store.VisitorList(r.Context(), statsstore.VisitorQuery().
		SetBot(statsstore.VALUE_YES).
		SetLimit(100000))
	if err != nil {
		api.Respond(w, r, api.Error("Failed to load bot visitors: "+err.Error()))
		return ""
	}

	// Aggregate by IP: hit count, last seen, and reasons (pattern matches).
	type ipAgg struct {
		IP       string
		Hits     int
		LastSeen string
		Reasons  map[string]*reasonAgg
	}
	agg := map[string]*ipAgg{}
	// dataCenterChecked tracks which IPs have already been checked for
	// data-center membership, since IsDataCenterIP is IP-level (not per-visit).
	dataCenterChecked := map[string]bool{}
	for _, v := range visitors {
		ip := v.GetIpAddress()
		if ip == "" {
			continue
		}
		a, ok := agg[ip]
		if !ok {
			a = &ipAgg{IP: ip, Reasons: map[string]*reasonAgg{}}
			agg[ip] = a
		}
		a.Hits++
		created := v.GetCreatedAt()
		if created > a.LastSeen {
			a.LastSeen = created
		}

		// Check visitor path against bot/malicious patterns to derive reasons.
		// Matching is case-insensitive (path is lowercased before comparison)
		// to catch paths like "/Robots.txt" that would otherwise slip through.
		path := v.GetPath()
		pathLower := strings.ToLower(path)
		for _, pattern := range botPagePatterns {
			if strings.Contains(pathLower, pattern) {
				reason, ok := a.Reasons[pattern]
				if !ok {
					reason = &reasonAgg{Pattern: pattern}
					a.Reasons[pattern] = reason
				}
				reason.Hits++
				if len(reason.Paths) < 3 && !slices.Contains(reason.Paths, path) {
					reason.Paths = append(reason.Paths, path)
				}
			}
		}

		// Check user agent for self-identifying bots.
		// The path is irrelevant — the signal is the UA string, not the URL.
		ua := v.GetUserAgent()
		if ua != "" && statsstore.IsBot(ua) {
			reason, ok := a.Reasons[shared.BotReasonUserAgent]
			if !ok {
				reason = &reasonAgg{Pattern: shared.BotReasonUserAgent}
				a.Reasons[shared.BotReasonUserAgent] = reason
			}
			reason.Hits++
		}

		// Check data-center IP (once per IP — it's IP-level, not per-visit).
		// The path is irrelevant — the signal is the IP range, not the URL.
		if !dataCenterChecked[ip] && statsstore.IsDataCenterIP(ip) {
			reason, ok := a.Reasons[shared.BotReasonDataCenterIP]
			if !ok {
				reason = &reasonAgg{Pattern: shared.BotReasonDataCenterIP}
				a.Reasons[shared.BotReasonDataCenterIP] = reason
			}
			reason.Hits++
		}
		dataCenterChecked[ip] = true

		// Check referrer spam.
		// The path is irrelevant — the signal is the referrer domain, not the URL.
		referrer := v.GetUserReferrer()
		if referrer != "" && statsstore.IsReferrerSpam(referrer) {
			reason, ok := a.Reasons[shared.BotReasonReferrerSpam]
			if !ok {
				reason = &reasonAgg{Pattern: shared.BotReasonReferrerSpam}
				a.Reasons[shared.BotReasonReferrerSpam] = reason
			}
			reason.Hits++
		}
	}

	list := make([]map[string]any, 0, len(agg))
	for _, a := range agg {
		list = append(list, map[string]any{
			FieldIP:       a.IP,
			FieldHits:     a.Hits,
			FieldLastSeen: a.LastSeen,
			FieldReasons:  reasonsToList(a.Reasons),
		})
	}
	sort.Slice(list, func(i, j int) bool {
		hi, _ := list[i][FieldHits].(int)
		hj, _ := list[j][FieldHits].(int)
		if hi != hj {
			return hi > hj
		}
		pi, _ := list[i][FieldIP].(string)
		pj, _ := list[j][FieldIP].(string)
		return pi < pj
	})

	api.Respond(w, r, api.SuccessWithData("Bot IPs loaded", map[string]any{
		FieldBots:  list,
		FieldTotal: len(list),
	}))
	return ""
}

// reasonsToList converts the reason aggregation map into a sorted slice for
// the API response, matching the format expected by the frontend.
func reasonsToList(reasons map[string]*reasonAgg) []map[string]any {
	if len(reasons) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(reasons))
	for _, r := range reasons {
		out = append(out, map[string]any{
			"pattern": r.Pattern,
			"hits":    r.Hits,
			"paths":   r.Paths,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		pi, _ := out[i]["pattern"].(string)
		pj, _ := out[j]["pattern"].(string)
		return pi < pj
	})
	return out
}
