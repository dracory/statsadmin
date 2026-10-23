package dashboard

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/dracory/api"
	"github.com/dracory/statsadmin/shared"
	"github.com/dracory/statsstore"
)

func (controller *dashboardController) handleExportCSV(w http.ResponseWriter, r *http.Request) string {
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
		SetLimit(50000)

	applyStoreConditions(query, reqBody.Conditions)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin dashboard: failed to load visitors for export", "error", err)
		api.Respond(w, r, api.Error("Failed to load visitor data: "+err.Error()))
		return ""
	}

	// Sanitize the period for use in the Content-Disposition filename to
	// prevent header injection (CRLF/quotes) via the request body.
	safePeriod := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			return r
		}
		return -1
	}, reqBody.Period)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=visitors_"+safePeriod+".csv")

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
