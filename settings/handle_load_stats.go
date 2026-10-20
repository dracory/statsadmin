package settings

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

// handleLoadStats returns record counts for the maintenance card.
// With no body or an empty older_than field it returns the total visitor
// record count. When older_than is provided it also returns how many
// records would be deleted by a prune operation for that date.
func (controller *settingsController) handleLoadStats(w http.ResponseWriter, r *http.Request) string {
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
		OlderThan string `json:"older_than"`
	}
	// Body is optional — ignore decode errors for empty bodies
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	// Total record count (excluding soft-deleted)
	totalCount, err := store.VisitorCount(r.Context(), statsstore.NewVisitorQuery())
	if err != nil {
		slog.Error("statsadmin settings: failed to count visitors", "error", err)
		api.Respond(w, r, api.Error(err.Error()))
		return ""
	}

	data := map[string]any{
		FieldTotalRecords: totalCount,
	}

	// If a date was provided, also count how many records are older than that date
	if reqBody.OlderThan != "" {
		olderCount, err := store.VisitorCount(r.Context(), statsstore.NewVisitorQuery().SetCreatedAtLte(reqBody.OlderThan))
		if err != nil {
			slog.Error("statsadmin settings: failed to count older visitors", "older_than", reqBody.OlderThan, "error", err)
			api.Respond(w, r, api.Error(err.Error()))
			return ""
		}
		data[FieldOlderCount] = olderCount
	}

	api.Respond(w, r, api.SuccessWithData("Record counts", data))
	return ""
}
