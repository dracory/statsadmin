package ipdetails

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dracory/statsadmin/shared"

	"github.com/dracory/api"
	"github.com/dracory/req"
	"github.com/dracory/statsstore"
)

// handleLoadBotReasons returns the computed bot reasons for a given IP. This
// is the AJAX endpoint used by the visitors and sessions pages to populate
// the bot-reasons modal when the user clicks a "Bot" badge. It fetches all
// visitor records for the IP and runs shared.ComputeBotReasons, which checks
// the same signals used at ingestion time (user-agent, data-center IP,
// referrer spam, bot/malicious paths).
func (controller *ipDetailsController) handleLoadBotReasons(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	ip := req.GetStringTrimmed(r, shared.ParamIP)
	if ip == "" {
		// Try JSON body if query param wasn't set.
		var body struct {
			IP string `json:"ip"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		ip = strings.TrimSpace(body.IP)
	}
	if ip == "" {
		api.Respond(w, r, api.Error("IP address is required"))
		return ""
	}

	ctx := r.Context()

	// Fetch all visitors for this IP. The store query supports IP filtering
	// via SetIPIn, but not all stores implement it; use the same broad-fetch
	// + in-memory filter approach as handleLoadPaths for consistency.
	visitors, err := store.VisitorList(ctx, statsstore.VisitorQuery().
		SetOrderBy("created_at").
		SetSortOrder("DESC").
		SetLimit(10000))
	if err != nil {
		slog.Error("ipdetails: failed to load visitors for bot reasons", "error", err)
		api.Respond(w, r, api.Error("Failed to load visitor data"))
		return ""
	}

	var ipVisitors []statsstore.VisitorInterface
	for _, v := range visitors {
		if v.GetIpAddress() == ip {
			ipVisitors = append(ipVisitors, v)
		}
	}

	if len(ipVisitors) == 0 {
		api.Respond(w, r, api.SuccessWithData("No visits found", map[string]any{
			FieldIP:         ip,
			FieldBotReasons: []any{},
			FieldVisitCount: 0,
		}))
		return ""
	}

	botReasons := shared.ComputeBotReasons(ipVisitors, nil)
	isBot := shared.IsBotVisitor(ipVisitors[0])

	api.Respond(w, r, api.SuccessWithData("Bot reasons loaded", map[string]any{
		FieldIP:         ip,
		FieldBotReasons: botReasons,
		FieldIsBot:      isBot,
		FieldVisitCount: len(ipVisitors),
	}))
	return ""
}
