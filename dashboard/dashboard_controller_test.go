package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dracory/statsadmin/shared"
	"github.com/dracory/statsstore"
)

func TestDashboardController_RendersPage(t *testing.T) {
	opts, layout, _ := shared.NewTestControllerOptions(t)

	controller := NewDashboardController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	w := httptest.NewRecorder()

	body := controller.Handler(w, req)

	if !strings.Contains(body, "Visitor Analytics") {
		t.Error("should render dashboard heading")
	}
	if !strings.Contains(layout.Body, "stats-dashboard-app") {
		t.Error("should render Vue.js app container in layout body")
	}
	if !strings.Contains(layout.Body, "vue.global.prod.js") {
		t.Error("should include Vue.js CDN")
	}

	hasFlagIcons := false
	for _, styleURL := range layout.StyleURLs {
		if styleURL == shared.FlagIconsCSS {
			hasFlagIcons = true
			break
		}
	}
	if !hasFlagIcons {
		t.Errorf("layout.StyleURLs should contain %s, got: %v", shared.FlagIconsCSS, layout.StyleURLs)
	}
}

func TestDashboardController_LoadDashboard_GetMethod(t *testing.T) {
	opts, _, _ := shared.NewTestControllerOptions(t)

	controller := NewDashboardController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats?action=load-dashboard", nil)
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(strings.ToLower(body), "error") {
		t.Error("GET on load-dashboard action should return error in response body")
	}
}

func TestDashboardController_LoadDashboard_Post(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitor(t, store, "1.2.3.4", "/docs", "US")
	shared.SeedVisitor(t, store, "5.6.7.8", "/", "GB")

	controller := NewDashboardController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-dashboard", strings.NewReader(`{"period":"all-time"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()
	if !strings.Contains(body, "total_visitors") {
		t.Error("should return total_visitors in response")
	}
	if !strings.Contains(body, "3") {
		t.Errorf("should report 3 total visitors, got: %s", body)
	}
}

func TestDashboardController_LoadDashboard_IncludeBotsToggle(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	// Seed regular visitor and bot visitor
	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitorWithBot(t, store, "9.9.9.9", "/bot", "US", "yes")

	controller := NewDashboardController(opts)

	// With no conditions (include bots)
	reqTrue := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-dashboard", strings.NewReader(`{"period":"all-time", "conditions": []}`))
	reqTrue.Header.Set("Content-Type", "application/json")
	wTrue := httptest.NewRecorder()
	_ = controller.Handler(wTrue, reqTrue)
	if !strings.Contains(wTrue.Body.String(), `"total_visitors":2`) {
		t.Errorf("expected 2 total visitors when include_bots=true, got: %s", wTrue.Body.String())
	}

	// With is_bot = no condition (exclude bots)
	reqFalse := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-dashboard", strings.NewReader(`{"period":"all-time", "conditions": [{"field":"is_bot","operator":"equals","value":"no"}]}`))
	reqFalse.Header.Set("Content-Type", "application/json")
	wFalse := httptest.NewRecorder()
	_ = controller.Handler(wFalse, reqFalse)
	if !strings.Contains(wFalse.Body.String(), `"total_visitors":1`) {
		t.Errorf("expected 1 total visitor when is_bot=no condition applied, got: %s", wFalse.Body.String())
	}
}

func TestDashboardController_LoadDashboard_BounceRate(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	// Session A: two page views 10 min apart (same fingerprint within the
	// gap window) => 600s duration, 2 pages. Session B: single view => bounce.
	// Expected: 1 bounce of 2 sessions => 50% bounce rate; avg duration
	// (600+0)/2 = 300s; pages per session (2+1)/2 = 1.5.
	for _, seed := range []struct{ path, createdAt string }{
		{"/a", "2020-01-01 10:00:00"},
		{"/b", "2020-01-01 10:10:00"},
	} {
		v := statsstore.NewVisitor().SetFingerprint("fp-1").SetIpAddress("1.2.3.4").SetPath(seed.path).SetCreatedAt(seed.createdAt)
		if err := store.VisitorCreate(context.Background(), v); err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
	v := statsstore.NewVisitor().SetFingerprint("fp-2").SetIpAddress("5.6.7.8").SetPath("/c").SetCreatedAt("2020-01-01 11:00:00")
	if err := store.VisitorCreate(context.Background(), v); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	controller := NewDashboardController(opts)

	req := httptest.NewRequest(http.MethodPost, "/admin/stats?action=load-dashboard", strings.NewReader(`{"period":"all-time"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)
	body := w.Body.String()

	if !strings.Contains(body, `"sessions":2`) {
		t.Errorf("expected 2 sessions, got: %s", body)
	}
	if !strings.Contains(body, `"bounce_rate":50`) {
		t.Errorf("expected 50%% bounce rate, got: %s", body)
	}
	if !strings.Contains(body, `"avg_session_duration":300`) {
		t.Errorf("expected 300s avg session duration, got: %s", body)
	}
	if !strings.Contains(body, `"pages_per_session":1.5`) {
		t.Errorf("expected 1.5 pages per session, got: %s", body)
	}
}
