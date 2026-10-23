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
		Conditions []condition `json:"conditions"`
	}
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

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
		api.Respond(w, r, api.Error("Failed to load data for export: "+err.Error()))
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
			shared.CSVSafe(v.GetID()),
			shared.CSVSafe(v.GetCreatedAt()),
			shared.CSVSafe(v.GetIpAddress()),
			shared.CSVSafe(v.GetFingerprint()),
			shared.CSVSafe(v.GetCountry()),
			shared.CSVSafe(controller.opts.CountryName(v.GetCountry())),
			shared.CSVSafe(v.GetPath()),
			shared.CSVSafe(v.GetUserBrowser()),
			shared.CSVSafe(v.GetUserOs()),
			shared.CSVSafe(v.GetUserDevice()),
			shared.CSVSafe(v.GetUserDeviceType()),
			shared.CSVSafe(v.GetUserReferrer()),
			shared.CSVSafe(botVal),
			shared.CSVSafe(v.GetThreat()),
		})
	}

	writer.Flush()
	return ""
}
