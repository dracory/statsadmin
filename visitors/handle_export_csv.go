package visitors

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dracory/api"
	"github.com/dracory/statsadmin/shared"
	"github.com/dracory/statsstore"
)

func (controller *visitorsController) handleExportCSV(w http.ResponseWriter, r *http.Request) string {
	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	var reqBody struct {
		Conditions []condition `json:"conditions"`
	}
	// Try POST body first
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	// If empty, try query param "filters" (for GET requests if needed, though buttons usually POST)
	if len(reqBody.Conditions) == 0 {
		if filters := r.URL.Query().Get("filters"); filters != "" {
			_ = json.Unmarshal([]byte(filters), &reqBody.Conditions)
		}
	}

	ctx := r.Context()
	storeConds, memConds := splitConditions(reqBody.Conditions)

	query := statsstore.VisitorQuery().
		SetOrderBy("created_at").
		SetSortOrder("DESC").
		SetLimit(50000) // Increase limit for export

	applyStoreConditions(query, storeConds)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin visitors: failed to load visitors for export", "error", err)
		api.Respond(w, r, api.Error("Failed to load data for export"))
		return ""
	}

	filtered := filterInMemory(visitors, memConds)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=visitors.csv")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"ID", "Created At", "IP Address", "Fingerprint", "Country", "Country Name",
		"Path", "Browser", "OS", "Device", "Device Type", "Referrer", "Is Bot", "Is Threat",
	})

	for _, v := range filtered {
		botVal := v.GetBot()
		if shared.IsBotVisitor(v) {
			botVal = statsstore.VALUE_YES
		}
		_ = writer.Write([]string{
			v.GetID(),
			v.GetCreatedAt(),
			v.GetIpAddress(),
			v.GetFingerprint(),
			v.GetCountry(),
			controller.opts.CountryName(v.GetCountry()),
			v.GetPath(),
			v.GetUserBrowser(),
			v.GetUserOs(),
			v.GetUserDevice(),
			v.GetUserDeviceType(),
			v.GetUserReferrer(),
			botVal,
			v.GetThreat(),
		})
	}

	writer.Flush()
	return ""
}
