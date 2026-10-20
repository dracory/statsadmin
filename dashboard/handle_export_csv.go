package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
	"github.com/dracory/statsadmin/shared"
)

func (controller *dashboardController) handleExportCSV(w http.ResponseWriter, r *http.Request) string {
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
		SetLimit(50000)

	applyStoreConditions(query, reqBody.Conditions)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin dashboard: failed to load visitors for export", "error", err)
		api.Respond(w, r, api.Error("Failed to load visitor data"))
		return ""
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=visitors_"+reqBody.Period+".csv")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"ID", "Created At", "IP Address", "Fingerprint", "Country", "Country Name",
		"Path", "Browser", "OS", "Device", "Device Type", "Referrer", "Is Bot", "Is Threat",
	})

	for _, v := range visitors {
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
