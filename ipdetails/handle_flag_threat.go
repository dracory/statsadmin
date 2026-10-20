package ipdetails

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

func (controller *ipDetailsController) handleFlagThreat(w http.ResponseWriter, r *http.Request) string {
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

	// Update all visitor records for this IP to threat='yes'.
	if err := flagIPVisitorsAsThreat(r.Context(), store, ip); err != nil {
		slog.Error("statsadmin ipdetails: failed to update visitor threat flag", "ip", ip, "error", err)
		api.Respond(w, r, api.Error(err.Error()))
		return ""
	}

	api.Respond(w, r, api.Success("IP flagged as threat"))
	return ""
}

// flagIPVisitorsAsThreat sets threat='yes' on all visitor records for the given IP.
func flagIPVisitorsAsThreat(ctx context.Context, store statsstore.StoreInterface, ip string) error {
	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetIPIn([]string{ip}).
		SetLimit(100000))
	if err != nil {
		return err
	}
	for _, v := range visitors {
		if v.GetThreat() == statsstore.VALUE_YES {
			continue
		}
		v.SetThreat(statsstore.VALUE_YES)
		if err := store.VisitorUpdate(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
