package ipdetails

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
)

func (controller *ipDetailsController) handleFlagBot(w http.ResponseWriter, r *http.Request) string {
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
	if _, err := netip.ParseAddr(ip); err != nil {
		api.Respond(w, r, api.Error("Invalid IP address"))
		return ""
	}

	// Set bot='yes' on all visitor records for this IP.
	count, err := shared.FlagIPVisitorsAsBot(r.Context(), store, ip)
	if err != nil {
		slog.Error("statsadmin ipdetails: failed to flag IP as bot", "ip", ip, "error", err)
		api.Respond(w, r, api.Error("Failed to flag IP: "+err.Error()))
		return ""
	}
	if count == 0 {
		api.Respond(w, r, api.Error("No visitor records found for this IP"))
		return ""
	}

	api.Respond(w, r, api.Success("IP flagged as bot"))
	return ""
}
