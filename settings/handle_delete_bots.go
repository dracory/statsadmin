package settings

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dracory/api"
	"github.com/dracory/statsstore"
)

func (controller *settingsController) handleDeleteBots(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	ctx := r.Context()

	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetBot(statsstore.VALUE_YES).
		SetLimit(100000))
	if err != nil {
		slog.Error("statsadmin settings: failed to load bot visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to load bot visitors"))
		return ""
	}

	deleted, failed := deleteVisitorsByID(ctx, store, visitors)

	api.Respond(w, r, api.SuccessWithData(
		fmt.Sprintf("Deleted %d bot visitor record(s)", deleted),
		map[string]any{
			FieldDeletedCount: deleted,
			"failed_count":    failed,
		},
	))
	return ""
}

func (controller *settingsController) handleDeleteThreats(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	ctx := r.Context()

	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetThreat(statsstore.VALUE_YES).
		SetLimit(100000))
	if err != nil {
		slog.Error("statsadmin settings: failed to load threat visitors", "error", err)
		api.Respond(w, r, api.Error("Failed to load threat visitors"))
		return ""
	}

	deleted, failed := deleteVisitorsByID(ctx, store, visitors)

	api.Respond(w, r, api.SuccessWithData(
		fmt.Sprintf("Deleted %d threat visitor record(s)", deleted),
		map[string]any{
			FieldDeletedCount: deleted,
			"failed_count":    failed,
		},
	))
	return ""
}

// deleteVisitorsByID deletes each visitor by ID and returns (deleted, failed) counts.
func deleteVisitorsByID(ctx context.Context, store statsstore.StoreInterface, visitors []statsstore.VisitorInterface) (int, int) {
	var deleted, failed int
	for _, v := range visitors {
		if err := store.VisitorDeleteByID(ctx, v.GetID()); err != nil {
			failed++
			slog.Error("statsadmin settings: failed to delete visitor", "id", v.GetID(), "error", err)
			continue
		}
		deleted++
	}
	return deleted, failed
}
