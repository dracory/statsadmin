package settings

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

	"github.com/dracory/api"
)

func (controller *settingsController) handleDeleteVisitors(w http.ResponseWriter, r *http.Request) string {
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
		IP        string `json:"ip"`
		OlderThan string `json:"older_than"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		api.Respond(w, r, api.Error("Invalid request body"))
		return ""
	}

	if ip := strings.TrimSpace(reqBody.IP); ip != "" {
		if _, err := netip.ParseAddr(ip); err != nil {
			api.Respond(w, r, api.Error("Invalid IP address"))
			return ""
		}

		count, err := store.VisitorDeleteByIP(r.Context(), ip)
		if err != nil {
			slog.Error("statsadmin settings: failed to delete visitors by IP", "ip", ip, "error", err)
			api.Respond(w, r, api.Error("Failed to delete visitors by IP: "+err.Error()))
			return ""
		}

		api.Respond(w, r, api.SuccessWithData(
			fmt.Sprintf("Deleted %d visitor record(s) for IP %s", count, ip),
			map[string]any{
				FieldDeletedCount: count,
				FieldIP:           ip,
			},
		))
		return ""
	}

	if reqBody.OlderThan != "" {
		count, err := store.VisitorDeleteOlderThan(r.Context(), reqBody.OlderThan)
		if err != nil {
			slog.Error("statsadmin settings: failed to delete visitors older than", "older_than", reqBody.OlderThan, "error", err)
			api.Respond(w, r, api.Error("Failed to delete visitors: "+err.Error()))
			return ""
		}

		api.Respond(w, r, api.SuccessWithData(
			fmt.Sprintf("Deleted %d visitor record(s) older than %s", count, reqBody.OlderThan),
			map[string]any{
				FieldDeletedCount: count,
			},
		))
		return ""
	}

	api.Respond(w, r, api.Error("IP address or OlderThan date is required"))
	return ""
}
