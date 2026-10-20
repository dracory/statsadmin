package sessions

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dracory/statsadmin/shared"
)

func TestSessionsController_RendersPage(t *testing.T) {
	opts, layout, _ := shared.NewTestControllerOptions(t)

	controller := NewSessionsController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	w := httptest.NewRecorder()

	body := controller.Handler(w, req)

	if !strings.Contains(body, "Sessions") {
		t.Error("should render sessions heading")
	}
	if !strings.Contains(layout.Body, "sessions-app") {
		t.Error("should render Vue.js app container")
	}
}

func TestSessionsController_LoadSessions_Post(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitor(t, store, "5.6.7.8", "/docs", "GB")

	controller := NewSessionsController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-sessions", strings.NewReader(`{"page":1,"per_page":25}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "sessions") {
		t.Error("should return sessions in response")
	}
	if !strings.Contains(body, `"total":2`) {
		t.Errorf("should report 2 total sessions, got: %s", body)
	}
}

func TestSessionsController_LoadSessions_IncludeBotsToggle(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitorWithBot(t, store, "9.9.9.9", "/bot", "US", "yes")

	controller := NewSessionsController(opts)

	// With no condition (include bots)
	reqTrue := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-sessions", strings.NewReader(`{"page":1,"per_page":25,"conditions":[]}`))
	reqTrue.Header.Set("Content-Type", "application/json")
	wTrue := httptest.NewRecorder()
	_ = controller.Handler(wTrue, reqTrue)
	if !strings.Contains(wTrue.Body.String(), `"total":2`) {
		t.Errorf("expected 2 total sessions when no bot condition, got: %s", wTrue.Body.String())
	}

	// With is_bot = no condition (exclude bots)
	reqFalse := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-sessions", strings.NewReader(`{"page":1,"per_page":25,"conditions":[{"field":"is_bot","operator":"equals","value":"no"}]}`))
	reqFalse.Header.Set("Content-Type", "application/json")
	wFalse := httptest.NewRecorder()
	_ = controller.Handler(wFalse, reqFalse)
	if !strings.Contains(wFalse.Body.String(), `"total":1`) {
		t.Errorf("expected 1 total session when is_bot=no condition applied, got: %s", wFalse.Body.String())
	}
}
