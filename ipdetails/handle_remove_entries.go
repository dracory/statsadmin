package ipdetails

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dracory/api"
)

// handleRemoveEntries deletes every visitor record (and its associated paths)
// for the given IP. This is a destructive operation and is gated behind a
// confirm dialog on the client side.
func (controller *ipDetailsController) handleRemoveEntries(w http.ResponseWriter, r *http.Request) string {
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

	ip := strings.TrimSpace(reqBody.IP)
	if ip == "" {
		api.Respond(w, r, api.Error("IP address is required"))
		return ""
	}

	count, err := store.VisitorDeleteByIP(r.Context(), ip)
	if err != nil {
		slog.Error("statsadmin ipdetails: failed to remove entries by IP", "ip", ip, "error", err)
		api.Respond(w, r, api.Error(err.Error()))
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
