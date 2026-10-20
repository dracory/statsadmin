package sessions

import (
	"encoding/csv"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

func (controller *sessionsController) handleExportCSV(w http.ResponseWriter, r *http.Request) string {
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
	query := statsstore.VisitorQuery().
		SetOrderBy("created_at").
		SetSortOrder("DESC").
		SetLimit(20000)

	applyStoreConditions(query, reqBody.Conditions)

	visitors, err := store.VisitorList(ctx, query)
	if err != nil {
		slog.Error("statsadmin sessions: failed to load visitors for export", "error", err)
		api.Respond(w, r, api.Error("Failed to load data for export"))
		return ""
	}

	filtered := filterInMemory(visitors, reqBody.Conditions)
	sessions := buildSessions(filtered, controller.opts)
	sortSessions(sessions)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=sessions.csv")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"IP Address", "Country", "Country Name", "Visit Count", "First Seen", "Last Seen",
		"Browser", "OS", "Device", "Device Type", "Is Bot", "Is Threat",
	})

	for _, s := range sessions {
		_ = writer.Write([]string{
			s.IP,
			s.Country,
			s.CountryName,
			strconv.Itoa(s.VisitCount),
			s.FirstSeen,
			s.LastSeen,
			s.Browser,
			s.OS,
			s.Device,
			s.DeviceType,
			strconv.FormatBool(s.IsBot),
			strconv.FormatBool(s.IsThreat),
		})
	}

	writer.Flush()
	return ""
}
