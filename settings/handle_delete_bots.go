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
		api.Respond(w, r, api.Error("Failed to load bot visitors: "+err.Error()))
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
		api.Respond(w, r, api.Error("Failed to load threat visitors: "+err.Error()))
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

// deleteVisitorsByID deletes the given visitors in batched DELETEs (500 IDs
// per statement) and returns (deleted, failed) counts.
func deleteVisitorsByID(ctx context.Context, store statsstore.StoreInterface, visitors []statsstore.VisitorInterface) (int, int) {
	const batchSize = 500

	ids := make([]string, 0, len(visitors))
	for _, v := range visitors {
		ids = append(ids, v.GetID())
	}

	var deleted, failed int
	for i := 0; i < len(ids); i += batchSize {
		end := min(i+batchSize, len(ids))
		n, err := store.VisitorDeleteByIDs(ctx, ids[i:end])
		if err != nil {
			failed += end - i
			slog.Error("statsadmin settings: failed to delete visitor batch", "count", end-i, "error", err)
			continue
		}
		deleted += int(n)
	}
	return deleted, failed
}
