package sessions

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

// sessionVisit is a single visit within a session.
type sessionVisit struct {
	Path     string `json:"path"`
	Created  string `json:"created_at"`
	Date     string `json:"created_at_date"`
	Time     string `json:"created_at_time"`
	Country  string `json:"country"`
	Device   string `json:"device"`
	IsBot    bool   `json:"is_bot"`
	IsThreat bool   `json:"is_threat"`
}

// session is a group of visits from a single IP.
type session struct {
	IP          string         `json:"ip"`
	Country     string         `json:"country"`
	CountryName string         `json:"country_name"`
	Browser     string         `json:"browser"`
	OS          string         `json:"os"`
	Device      string         `json:"device"`
	DeviceType  string         `json:"device_type"`
	VisitCount  int            `json:"visit_count"`
	FirstSeen   string         `json:"first_seen"`
	LastSeen    string         `json:"last_seen"`
	IsBot       bool           `json:"is_bot"`
	IsThreat    bool           `json:"is_threat"`
	Visits      []sessionVisit `json:"visits"`
}

// condition is a single filter condition sent from the frontend.
type condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func (controller *sessionsController) handleLoadSessions(w http.ResponseWriter, r *http.Request) string {
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
		Page       int         `json:"page"`
		PerPage    int         `json:"per_page"`
		Conditions []condition `json:"conditions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Respond(w, r, api.Error("Invalid request body"))
		return ""
	}

	if reqBody.Page < 1 {
		reqBody.Page = 1
	}
	if reqBody.PerPage <= 0 || reqBody.PerPage > shared.MaxPageSize {
		reqBody.PerPage = shared.DefaultPageSize
	}

	ctx := r.Context()

	// Build the store query from store-queryable conditions.
	query := statsstore.VisitorQuery().
		SetOrderBy("created_at").
		SetSortOrder("DESC").
		SetLimit(10000)

	applyStoreConditions(query, reqBody.Conditions)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin sessions: failed to load visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to load sessions: "+err.Error()))
		return ""
	}

	// Apply in-memory conditions and build sessions.
	filtered := filterInMemory(visitors, reqBody.Conditions)
	sessions := buildSessions(filtered, controller.opts)

	// Sort sessions by most recent activity.
	sortSessions(sessions)

	// Paginate by session.
	total := len(sessions)
	start := (reqBody.Page - 1) * reqBody.PerPage
	end := start + reqBody.PerPage
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pageSlice := sessions[start:end]

	totalPages := total / reqBody.PerPage
	if total%reqBody.PerPage != 0 {
		totalPages++
	}

	api.Respond(w, r, api.SuccessWithData("Sessions loaded", map[string]any{
		FieldSessions:   pageSlice,
		FieldTotal:      total,
		FieldPage:       reqBody.Page,
		FieldPerPage:    reqBody.PerPage,
		FieldTotalPages: totalPages,
	}))
	return ""
}

func hasBotCondition(conds []condition) bool {
	for _, c := range conds {
		if c.Field == "is_bot" {
			return true
		}
	}
	return false
}

// applyStoreConditions applies store-queryable conditions to the query.
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
		}
	}
}

// filterInMemory applies conditions that the store cannot handle (IP, Browser, OS).
func filterInMemory(visitors []statsstore.VisitorInterface, conds []condition) []statsstore.VisitorInterface {
	if len(conds) == 0 {
		return visitors
	}

	byField := map[string][]condition{}
	for _, c := range conds {
		if c.Field == "ip" || c.Field == "browser" || c.Field == "os" {
			byField[c.Field] = append(byField[c.Field], c)
		}
	}
	if len(byField) == 0 {
		return visitors
	}

	result := make([]statsstore.VisitorInterface, 0, len(visitors))
	for _, v := range visitors {
		if matchesAllFields(v, byField) {
			result = append(result, v)
		}
	}
	return result
}

func matchesAllFields(v statsstore.VisitorInterface, byField map[string][]condition) bool {
	for field, conds := range byField {
		if !matchesAnyInField(v, field, conds) {
			return false
		}
	}
	return true
}

func matchesAnyInField(v statsstore.VisitorInterface, field string, conds []condition) bool {
	for _, c := range conds {
		val := strings.TrimSpace(c.Value)
		if val == "" {
			continue
		}
		switch field {
		case "ip":
			if v.GetIpAddress() == val {
				return true
			}
		case "browser":
			if strings.EqualFold(v.GetUserBrowser(), val) {
				return true
			}
		case "os":
			if strings.EqualFold(v.GetUserOs(), val) {
				return true
			}
		}
	}
	return false
}

// buildSessions groups visits by IP. Each IP becomes one session containing all visits.
func buildSessions(visitors []statsstore.VisitorInterface, opts shared.ControllerOptions) []session {
	byIP := map[string]*session{}
	for _, v := range visitors {
		ip := v.GetIpAddress()
		isBot := shared.IsBotVisitor(v)
		isThreat := v.GetThreat() == statsstore.VALUE_YES
		s, ok := byIP[ip]
		if !ok {
			s = &session{
				IP:          ip,
				Country:     v.GetCountry(),
				CountryName: opts.CountryName(v.GetCountry()),
				Browser:     v.GetUserBrowser(),
				OS:          v.GetUserOs(),
				Device:      v.GetUserDevice(),
				DeviceType:  v.GetUserDeviceType(),
				IsBot:       isBot,
				IsThreat:    isThreat,
				Visits:      []sessionVisit{},
			}
			byIP[ip] = s
		}
		s.Visits = append(s.Visits, newSessionVisit(v, isBot, isThreat))
	}

	// Build slice and compute session-level metadata.
	sessions := make([]session, 0, len(byIP))
	for _, s := range byIP {
		// Visits are in DESC created_at order already. Keep them sorted.
		sort.Slice(s.Visits, func(i, j int) bool {
			return s.Visits[i].Created > s.Visits[j].Created
		})
		s.VisitCount = len(s.Visits)
		s.FirstSeen = s.Visits[len(s.Visits)-1].Created
		s.LastSeen = s.Visits[0].Created

		sessions = append(sessions, *s)
	}
	return sessions
}

func newSessionVisit(v statsstore.VisitorInterface, isBot bool, isThreat bool) sessionVisit {
	created := v.GetCreatedAt()
	date, time := splitDateTime(created)
	return sessionVisit{
		Path:     v.GetPath(),
		Created:  created,
		Date:     date,
		Time:     time,
		Country:  v.GetCountry(),
		Device:   v.GetUserDevice(),
		IsBot:    isBot,
		IsThreat: isThreat,
	}
}

// splitDateTime splits a "YYYY-MM-DD HH:MM:SS" string into date and time.
func splitDateTime(value string) (string, string) {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// sortSessions sorts sessions by last seen, most recent first.
func sortSessions(sessions []session) {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastSeen > sessions[j].LastSeen
	})
}
