package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dracory/statsadmin/shared"
)

// apiResponse mirrors github.com/dracory/api.Response for test parsing.
type apiResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

func callAction(t *testing.T, opts shared.ControllerOptions, method, action string, jsonBody string) apiResponse {
	t.Helper()
	controller := NewSettingsController(opts)

	var bodyReader *strings.Reader
	if jsonBody != "" {
		bodyReader = strings.NewReader(jsonBody)
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, "/admin/stats?action="+action, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	_ = controller.Handler(w, req)

	var resp apiResponse
	_ = json.Unmarshal([]byte(w.Body.String()), &resp)
	return resp
}

func TestSettingsController_RendersPage(t *testing.T) {
	opts, layout, _ := shared.NewTestControllerOptions(t)

	controller := NewSettingsController(opts)

	req := httptest.NewRequest(http.MethodGet, "/admin/stats", nil)
	w := httptest.NewRecorder()

	body := controller.Handler(w, req)

	if !strings.Contains(body, "Stats Settings") {
		t.Error("should render settings heading")
	}
	if !strings.Contains(layout.Body, "settings-app") {
		t.Error("should render Vue.js app container")
	}
}

func TestSettingsController_LoadIPs(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	_ = store.ExcludedIPAdd(nil, "1.2.3.4")

	resp := callAction(t, opts, http.MethodPost, actionLoadIPs, "")

	if resp.Status != "success" {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	if resp.Data["total"].(float64) != 1 {
		t.Errorf("expected 1 IP, got %v", resp.Data["total"])
	}
}

func TestSettingsController_AddIP(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	resp := callAction(t, opts, http.MethodPost, actionAddIP, `{"ip":"10.0.0.1"}`)

	if resp.Status != "success" {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	ips, _ := store.ExcludedIPList(nil)
	if len(ips) != 1 || ips[0] != "10.0.0.1" {
		t.Errorf("expected [10.0.0.1], got %v", ips)
	}
}

func TestSettingsController_RemoveIP(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	_ = store.ExcludedIPAdd(nil, "10.0.0.1")

	resp := callAction(t, opts, http.MethodPost, actionRemoveIP, `{"ip":"10.0.0.1"}`)

	if resp.Status != "success" {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	ips, _ := store.ExcludedIPList(nil)
	if len(ips) != 0 {
		t.Errorf("expected 0 IPs after removal, got %v", ips)
	}
}

func TestSettingsController_LoadBots(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/robots.txt", "US")

	// Flag the visitor as a bot.
	_, _ = shared.FlagIPVisitorsAsBot(nil, store, "1.2.3.4")

	resp := callAction(t, opts, http.MethodPost, actionLoadBots, "")

	if resp.Status != "success" {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
}

func TestSettingsController_AddBot(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")

	resp := callAction(t, opts, http.MethodPost, actionAddBot, `{"ip":"1.2.3.4"}`)

	if resp.Status != "success" {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
}

func TestSettingsController_DeleteVisitors(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	shared.SeedVisitor(t, store, "1.2.3.4", "/", "US")
	shared.SeedVisitor(t, store, "1.2.3.4", "/docs", "US")

	resp := callAction(t, opts, http.MethodPost, actionDeleteVisitors, `{"ip":"1.2.3.4"}`)

	if resp.Status != "success" {
		t.Errorf("expected success, got %s: %s", resp.Status, resp.Message)
	}
	deleted, _ := resp.Data["deleted_count"].(float64)
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %v", resp.Data["deleted_count"])
	}
}

func TestSettingsController_IdentifyBots_WlwManifest(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	// Seed a visitor requesting /blog/wp-includes/wlwmanifest.xml
	shared.SeedVisitor(t, store, "5.6.7.8", "/blog/wp-includes/wlwmanifest.xml", "US")

	resp := callAction(t, opts, http.MethodPost, actionIdentifyBots, "")

	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Message)
	}

	autoAdded, ok := resp.Data["auto_added"].([]interface{})
	if !ok || len(autoAdded) == 0 {
		t.Fatalf("expected 5.6.7.8 to be auto-added as a bot, got %v", resp.Data["auto_added"])
	}

	found := false
	for _, ip := range autoAdded {
		if ip == "5.6.7.8" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected IP 5.6.7.8 in auto_added list, got %v", autoAdded)
	}
}

func TestSettingsController_IdentifyBots_WordPressProbing(t *testing.T) {
	opts, _, store := shared.NewTestControllerOptions(t)

	// Seed a visitor requesting /WordPress directory probe
	shared.SeedVisitor(t, store, "103.45.230.198", "/WordPress", "US")

	resp := callAction(t, opts, http.MethodPost, actionIdentifyBots, "")

	if resp.Status != "success" {
		t.Fatalf("expected success, got %s: %s", resp.Status, resp.Message)
	}

	autoAdded, ok := resp.Data["auto_added"].([]interface{})
	if !ok || len(autoAdded) == 0 {
		t.Fatalf("expected 103.45.230.198 to be auto-added as a bot, got %v", resp.Data["auto_added"])
	}

	found := false
	for _, ip := range autoAdded {
		if ip == "103.45.230.198" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected IP 103.45.230.198 in auto_added list, got %v", autoAdded)
	}
}
