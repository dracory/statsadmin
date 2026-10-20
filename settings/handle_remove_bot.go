package settings

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
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

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	// Set bot='no' on all visitor records for this IP.
	if err := unflagIPVisitorsAsBot(r.Context(), store, ip); err != nil {
		slog.Error("statsadmin settings: failed to clear visitor bot flag", "ip", ip, "error", err)
		api.Respond(w, r, api.Error(err.Error()))
		return ""
	}

	api.Respond(w, r, api.Success("IP removed from bot list"))
	return ""
}

// unflagIPVisitorsAsBot sets bot='no' on all visitor records for the given IP.
func unflagIPVisitorsAsBot(ctx context.Context, store statsstore.StoreInterface, ip string) error {
	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetIPIn([]string{ip}).
		SetLimit(100000))
	if err != nil {
		return err
	}
	for _, v := range visitors {
		if v.GetBot() == statsstore.VALUE_NO {
			continue
		}
		v.SetBot(statsstore.VALUE_NO)
		if err := store.VisitorUpdate(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
