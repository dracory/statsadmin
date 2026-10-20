package settings

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dracory/api"
)

func (controller *settingsController) handleAddIP(w http.ResponseWriter, r *http.Request) string {
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
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Respond(w, r, api.Error("Invalid request body"))
		return ""
	}

	if reqBody.IP == "" {
		api.Respond(w, r, api.Error("IP address is required"))
		return ""
	}

	if err := store.ExcludedIPAdd(r.Context(), reqBody.IP); err != nil {
		slog.Error("statsadmin settings: failed to add excluded IP", "ip", reqBody.IP, "error", err)
		api.Respond(w, r, api.Error("Failed to add IP"))
		return ""
	}

	api.Respond(w, r, api.Success("IP added to exclusion list"))
	return ""
}
