package ipdetails

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dracory/statsadmin/shared"
)

func TestIPDetailsController_RendersPage(t *testing.T) {
	opts, layout, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")

	controller := NewIPDetailsController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats?controller=ip-details&ip=1.2.3.4", nil)
	w := httptest.NewRecorder()

	body := controller.Handler(w, req)

	if !strings.Contains(body, "IP Details: 1.2.3.4") {
		t.Errorf("should render IP details heading, got: %s", body)
	}
	if !strings.Contains(layout.Body, "ip-details-app") {
		t.Error("should render Vue.js app container")
	}
}

func TestIPDetailsController_NoIP(t *testing.T) {
	opts, _, _ := shared.NewTestControllerOptions(t)

	controller := NewIPDetailsController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats?controller=ip-details", nil)
	w := httptest.NewRecorder()

	body := controller.Handler(w, req)

	// Should redirect (FlashError is nil so falls back to http.Redirect → 303).
	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect 303 when no IP, got %d (body: %s)", w.Code, body)
	}
}

func TestIPDetailsController_LoadPaths_Post(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitor(t, store, "1.2.3.4", "/docs", "US")
	shared.SeedVisitor(t, store, "5.6.7.8", "/", "GB")

	controller := NewIPDetailsController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-paths&ip=1.2.3.4", strings.NewReader(`{"ip":"1.2.3.4"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "details") {
		t.Error("should return details in response")
	}
	if !strings.Contains(body, "1.2.3.4") {
		t.Error("should contain the IP in response")
	}
}

func TestIPDetailsController_FlagBot_Post(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")

	controller := NewIPDetailsController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=flag-bot", strings.NewReader(`{"ip":"1.2.3.4"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "success") {
		t.Errorf("expected success, got: %s", body)
	}
}

func TestIPDetailsController_RemoveEntries_Post(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitor(t, store, "1.2.3.4", "/docs", "US")

	controller := NewIPDetailsController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=remove-entries", strings.NewReader(`{"ip":"1.2.3.4"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "success") {
		t.Errorf("expected success, got: %s", body)
	}
	if !strings.Contains(body, "2") {
		t.Errorf("should report 2 deleted, got: %s", body)
	}
}

func TestIPDetailsController_LoadPaths_IncludeBotsToggle(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitorWithBot(t, store, "1.2.3.4", "/bot", "US", "yes")

	controller := NewIPDetailsController(opts)

	// With no condition (include bots)
	reqTrue := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-paths&ip=1.2.3.4", strings.NewReader(`{"ip":"1.2.3.4","conditions":[]}`))
	reqTrue.Header.Set("Content-Type", "application/json")
	wTrue := httptest.NewRecorder()
	_ = controller.Handler(wTrue, reqTrue)
	if !strings.Contains(wTrue.Body.String(), `"visit_count":2`) {
		t.Errorf("expected visit_count: 2 when no bot condition, got: %s", wTrue.Body.String())
	}

	// With is_bot = no condition (exclude bots)
	reqFalse := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-paths&ip=1.2.3.4", strings.NewReader(`{"ip":"1.2.3.4","conditions":[{"field":"is_bot","operator":"equals","value":"no"}]}`))
	reqFalse.Header.Set("Content-Type", "application/json")
	wFalse := httptest.NewRecorder()
	_ = controller.Handler(wFalse, reqFalse)
	if !strings.Contains(wFalse.Body.String(), `"visit_count":1`) {
		t.Errorf("expected visit_count: 1 when is_bot=no condition applied, got: %s", wFalse.Body.String())
	}
}
