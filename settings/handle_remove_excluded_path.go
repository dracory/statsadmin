package settings

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dracory/api"
)

func (controller *settingsController) handleRemoveExcludedPath(w http.ResponseWriter, r *http.Request) string {
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
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Respond(w, r, api.Error("Invalid request body"))
		return ""
	}

	path := strings.TrimSpace(reqBody.Path)
	if path == "" {
		api.Respond(w, r, api.Error("Path prefix is required"))
		return ""
	}

	if err := store.ExcludedPathRemove(r.Context(), path); err != nil {
		slog.Error("statsadmin settings: failed to remove excluded path", "path", path, "error", err)
		api.Respond(w, r, api.Error("Failed to remove path"))
		return ""
	}

	api.Respond(w, r, api.Success("Path removed from exclusion list"))
	return ""
}
