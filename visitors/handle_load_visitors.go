package visitors

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

// condition is a single filter condition sent from the frontend.
type condition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func (controller *visitorsController) handleLoadVisitors(w http.ResponseWriter, r *http.Request) string {
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

	// Split conditions into store-queryable vs in-memory.
	storeConds, memConds := splitConditions(reqBody.Conditions)

	// Build the store query from store-queryable conditions.
	query := statsstore.VisitorQuery().
		SetOrderBy("created_at").
		SetSortOrder("DESC").
		SetLimit(10000) // fetch a large batch, then paginate in-memory if needed

	applyStoreConditions(query, storeConds)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin visitors: failed to load visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to load visitors: "+err.Error()))
		return ""
	}

	// Apply in-memory conditions (IP, Browser, OS).
	filtered := filterInMemory(visitors, memConds)

	// Paginate the filtered result.
	total := int64(len(filtered))
	start := (reqBody.Page - 1) * reqBody.PerPage
	end := start + reqBody.PerPage
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	pageSlice := filtered[start:end]

	list := make([]map[string]any, 0, len(pageSlice))
	for _, v := range pageSlice {
		list = append(list, map[string]any{
			FieldID:          v.GetID(),
			FieldIP:          v.GetIpAddress(),
			FieldCountry:     v.GetCountry(),
			FieldCountryName: controller.opts.CountryName(v.GetCountry()),
			FieldPath:        v.GetPath(),
			FieldBrowser:     v.GetUserBrowser(),
			FieldOS:          v.GetUserOs(),
			FieldDevice:      v.GetUserDevice(),
			FieldDeviceType:  v.GetUserDeviceType(),
			FieldCreatedAt:   v.GetCreatedAt(),
			FieldUserAgent:   v.GetUserAgent(),
			FieldIsBot:       shared.IsBotVisitor(v),
			FieldIsThreat:    v.GetThreat() == statsstore.VALUE_YES,
		})
	}

	totalPages := int(total) / reqBody.PerPage
	if int(total)%reqBody.PerPage != 0 {
		totalPages++
	}

	api.Respond(w, r, api.SuccessWithData("Visitors loaded", map[string]any{
		FieldVisitors:   list,
		FieldTotal:      total,
		FieldPage:       reqBody.Page,
		FieldPerPage:    reqBody.PerPage,
		FieldTotalPages: totalPages,
	}))
	return ""
}

func hasBotCondition(conds []condition) bool {
	for _, c := range conds {
		if c.Field == CondFieldIsBot {
			return true
		}
	}
	return false
}

// splitConditions separates conditions into store-queryable and in-memory.
func splitConditions(conds []condition) (storeConds, memConds []condition) {
	for _, c := range conds {
		switch c.Field {
		case CondFieldCountry, CondFieldDeviceType, CondFieldPathCont,
			CondFieldPathExact, CondFieldDateFrom, CondFieldDateTo, CondFieldIsBot, CondFieldIsThreat:
			storeConds = append(storeConds, c)
		case CondFieldIP, CondFieldBrowser, CondFieldOS:
			memConds = append(memConds, c)
		}
	}
	return
}

// applyStoreConditions applies store-queryable conditions to the query.
// If the same field appears multiple times, the last one wins (store query
// only supports one value per field). For multiple IP/country/etc conditions
// the in-memory path handles the rest.
func applyStoreConditions(query statsstore.VisitorQueryInterface, conds []condition) {
	for _, c := range conds {
		val := strings.TrimSpace(c.Value)
		if val == "" {
			continue
		}
		switch c.Field {
		case CondFieldCountry:
			query.SetCountry(val)
		case CondFieldDeviceType:
			query.SetDeviceType(val)
		case CondFieldPathCont:
			query.SetPathContains(val)
		case CondFieldPathExact:
			query.SetPathExact(val)
		case CondFieldDateFrom:
			query.SetCreatedAtGte(val + " 00:00:00")
		case CondFieldDateTo:
			query.SetCreatedAtLte(val + " 23:59:59")
		case CondFieldIsBot:
			if strings.EqualFold(val, CondValueBotYes) {
				query.SetBot(statsstore.VALUE_YES)
			} else if strings.EqualFold(val, CondValueBotNo) {
				query.SetBot(statsstore.VALUE_NO)
			}
		case CondFieldIsThreat:
			if strings.EqualFold(val, CondValueBotYes) {
				query.SetThreat(statsstore.VALUE_YES)
			} else if strings.EqualFold(val, CondValueBotNo) {
				query.SetThreat(statsstore.VALUE_NO)
			}
		}
	}
}

// filterInMemory applies conditions that the store query cannot handle
// (IP, Browser, OS). Multiple conditions on the same field are OR-combined
// within the field, and fields are AND-combined — so "IP=1 OR IP=2" AND
// "Browser=Chrome" works.
func filterInMemory(visitors []statsstore.VisitorInterface, conds []condition) []statsstore.VisitorInterface {
	if len(conds) == 0 {
		return visitors
	}

	// Group conditions by field for OR-within-field, AND-across-fields.
	byField := map[string][]condition{}
	for _, c := range conds {
		byField[c.Field] = append(byField[c.Field], c)
	}

	result := make([]statsstore.VisitorInterface, 0, len(visitors))
	for _, v := range visitors {
		if matchesAllFields(v, byField) {
			result = append(result, v)
		}
	}
	return result
}

// matchesAllFields returns true if the visitor matches at least one condition
// in every field group (AND across fields, OR within a field).
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
		case CondFieldIP:
			if v.GetIpAddress() == val {
				return true
			}
		case CondFieldBrowser:
			if strings.EqualFold(v.GetUserBrowser(), val) {
				return true
			}
		case CondFieldOS:
			if strings.EqualFold(v.GetUserOs(), val) {
				return true
			}
		}
	}
	return false
}
