package visitors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dracory/statsadmin/shared"
)

func TestVisitorsController_RendersPage(t *testing.T) {
	opts, layout, _ := shared.NewTestControllerOptions(t)

	controller := NewVisitorsController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	w := httptest.NewRecorder()

	body := controller.Handler(w, req)

	if !strings.Contains(body, "Visitors") {
		t.Error("should render visitors heading")
	}
	if !strings.Contains(layout.Body, "visitors-app") {
		t.Error("should render Vue.js app container")
	}
}

func TestVisitorsController_LoadVisitors_Post(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitor(t, store, "5.6.7.8", "/docs", "GB")

	controller := NewVisitorsController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-visitors", strings.NewReader(`{"page":1,"per_page":25}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "visitors") {
		t.Error("should return visitors in response")
	}
	if !strings.Contains(body, "2") {
		t.Errorf("should report 2 total visitors, got: %s", body)
	}
}

func TestVisitorsController_LoadVisitors_GetMethod(t *testing.T) {
	opts, _, _ := shared.NewTestControllerOptions(t)

	controller := NewVisitorsController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats?action=load-visitors", nil)
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(strings.ToLower(body), "error") {
		t.Error("GET on load-visitors should return error")
	}
}

func TestVisitorsController_LoadVisitors_IncludeBotsToggle(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitorWithBot(t, store, "9.9.9.9", "/bot", "US", "yes")

	controller := NewVisitorsController(opts)

	// With no condition (include bots)
	reqTrue := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-visitors", strings.NewReader(`{"page":1,"per_page":25,"conditions":[]}`))
	reqTrue.Header.Set("Content-Type", "application/json")
	wTrue := httptest.NewRecorder()
	_ = controller.Handler(wTrue, reqTrue)
	if !strings.Contains(wTrue.Body.String(), `"total":2`) {
		t.Errorf("expected 2 total visitors when no bot condition, got: %s", wTrue.Body.String())
	}

	// With is_bot = no condition (exclude bots)
	reqFalse := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-visitors", strings.NewReader(`{"page":1,"per_page":25,"conditions":[{"field":"is_bot","operator":"equals","value":"no"}]}`))
	reqFalse.Header.Set("Content-Type", "application/json")
	wFalse := httptest.NewRecorder()
	_ = controller.Handler(wFalse, reqFalse)
	if !strings.Contains(wFalse.Body.String(), `"total":1`) {
		t.Errorf("expected 1 total visitor when is_bot=no condition applied, got: %s", wFalse.Body.String())
	}
}

func TestVisitorsController_LoadVisitors_DeviceBotIsBotTrue(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	// Seed visitor with device_type = "bot" but bot column = "no"
	v := shared.SeedVisitorWithBot(t, store, "87.250.224.7", "/robots.txt", "", "no")
	v.SetUserDeviceType("bot")
	_ = store.VisitorUpdate(nil, v)

	controller := NewVisitorsController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-visitors", strings.NewReader(`{"page":1,"per_page":25}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()

	if !strings.Contains(body, `"is_bot":true`) {
		t.Errorf("expected is_bot:true when device_type is 'bot', got response: %s", body)
	}

	// Also verify that filtering by is_bot = no excludes this device='bot' visitor
	reqFilter := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-visitors", strings.NewReader(`{"page":1,"per_page":25,"conditions":[{"field":"is_bot","operator":"equals","value":"no"}]}`))
	reqFilter.Header.Set("Content-Type", "application/json")
	wFilter := httptest.NewRecorder()

	_ = controller.Handler(wFilter, reqFilter)
	if !strings.Contains(wFilter.Body.String(), `"total":0`) {
		t.Errorf("expected 0 visitors when filtering is_bot=no for device_type='bot', got response: %s", wFilter.Body.String())
	}
}
