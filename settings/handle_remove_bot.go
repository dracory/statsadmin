package settings

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

func (controller *settingsController) handleRemoveBot(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
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
	if _, err := netip.ParseAddr(ip); err != nil {
		api.Respond(w, r, api.Error("Invalid IP address"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	// Set bot='no' on all visitor records for this IP.
	if _, err := store.VisitorSetBotByIP(r.Context(), ip, statsstore.VALUE_NO); err != nil {
		slog.Error("statsadmin settings: failed to clear visitor bot flag", "ip", ip, "error", err)
		api.Respond(w, r, api.Error("Failed to remove bot IP: "+err.Error()))
		return ""
	}

	api.Respond(w, r, api.Success("IP removed from bot list"))
	return ""
}
