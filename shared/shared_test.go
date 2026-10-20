package shared

import (
	"context"
	"testing"

	"github.com/dracory/statsstore"
)

func TestResolvePeriod_Default(t *testing.T) {
	bounds := ResolvePeriod("")
	if bounds.From == "" || bounds.To == "" {
		t.Error("default period should produce non-empty bounds")
	}
	if bounds.Label == "" {
		t.Error("default period should have a label")
	}
}

func TestResolvePeriod_AllTime(t *testing.T) {
	bounds := ResolvePeriod(PeriodAllTime)
	if bounds.From != "1970-01-01 00:00:00" {
		t.Errorf("all-time From should be epoch, got %s", bounds.From)
	}
	if bounds.Label != "All time" {
		t.Errorf("expected 'All time', got %s", bounds.Label)
	}
}

func TestPeriodOptions_Length(t *testing.T) {
	opts := PeriodOptions()
	if len(opts) != 6 {
		t.Errorf("expected 6 period options, got %d", len(opts))
	}
}

func TestAggregateCounts(t *testing.T) {
	visitors := []VisitorLike{
		mockVisitor{country: "US"},
		mockVisitor{country: "US"},
		mockVisitor{country: "GB"},
		mockVisitor{country: ""}, // empty → grouped under "—"
	}
	counts := AggregateCounts(visitors, func(v VisitorLike) string { return v.GetCountry() })

	if len(counts) != 3 {
		t.Errorf("expected 3 entries, got %d", len(counts))
	}
	// US should be first (count=2, highest)
	if counts[0].Label != "US" || counts[0].Count != 2 {
		t.Errorf("expected US count=2 first, got %+v", counts[0])
	}
}

func TestTopN(t *testing.T) {
	in := []CountEntry{
		{Label: "a", Count: 3},
		{Label: "b", Count: 2},
		{Label: "c", Count: 1},
	}
	out := TopN(in, 2)
	if len(out) != 2 {
		t.Errorf("expected 2, got %d", len(out))
	}
	if out[0].Label != "a" {
		t.Errorf("expected 'a' first, got %s", out[0].Label)
	}
}

func TestFlagIPVisitorsAsBot(t *testing.T) {
	store := NewTestStore(t)

	v1 := statsstore.NewVisitor().SetIpAddress("1.2.3.4").SetPath("/")
	v2 := statsstore.NewVisitor().SetIpAddress("1.2.3.4").SetPath("/docs")
	v3 := statsstore.NewVisitor().SetIpAddress("5.6.7.8").SetPath("/")

	for _, v := range []statsstore.VisitorInterface{v1, v2, v3} {
		if err := store.VisitorCreate(context.Background(), v); err != nil {
			t.Fatalf("failed to seed: %v", err)
		}
	}

	count, err := FlagIPVisitorsAsBot(context.Background(), store, "1.2.3.4")
	if err != nil {
		t.Fatalf("FlagIPVisitorsAsBot failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 visitors flagged, got %d", count)
	}

	// Verify the other IP was not flagged.
	visitors, _ := store.VisitorList(context.Background(), statsstore.VisitorQuery().SetIPIn([]string{"5.6.7.8"}))
	for _, v := range visitors {
		if v.GetBot() == statsstore.VALUE_YES {
			t.Error("5.6.7.8 should not be flagged as bot")
		}
	}
}

func TestNewLinks(t *testing.T) {
	l := NewLinks("/admin/stats")
	if l.Base() != "/admin/stats" {
		t.Errorf("expected /admin/stats, got %s", l.Base())
	}

	url := l.Dashboard(map[string]string{"action": "load-dashboard"})
	if url == "" {
		t.Error("Dashboard URL should not be empty")
	}
	if !contains(url, "controller=dashboard") {
		t.Errorf("URL should contain controller=dashboard, got %s", url)
	}
	if !contains(url, "action=load-dashboard") {
		t.Errorf("URL should contain action=load-dashboard, got %s", url)
	}
}

func TestNewLinks_EmptyBaseURL(t *testing.T) {
	l := NewLinks("")
	if l.Base() != "/" {
		t.Errorf("empty base URL should default to /, got %s", l.Base())
	}
}

func TestCountryNameResolver_NilCallback(t *testing.T) {
	r := NewCountryNameResolver(nil)
	if name := r.Name(context.Background(), "US"); name != "US" {
		t.Errorf("nil callback should return raw code, got %s", name)
	}
}

func TestCountryNameResolver_WithCallback(t *testing.T) {
	r := NewCountryNameResolver(func(iso2 string) (string, error) {
		if iso2 == "US" {
			return "United States", nil
		}
		return iso2, nil
	})
	if name := r.Name(context.Background(), "US"); name != "United States" {
		t.Errorf("expected 'United States', got %s", name)
	}
}

func TestCountryNameResolver_EmptyCode(t *testing.T) {
	r := NewCountryNameResolver(nil)
	if name := r.Name(context.Background(), ""); name != "—" {
		t.Errorf("empty code should return '—', got %s", name)
	}
}

func TestCountryNameResolver_UnknownCodeEmptyReturn(t *testing.T) {
	r := NewCountryNameResolver(func(iso2 string) (string, error) {
		return "", nil
	})
	if name := r.Name(context.Background(), "XX"); name != "XX" {
		t.Errorf("expected raw code XX when lookup returns empty string, got %s", name)
	}
}

func TestControllerOptions_CountryName(t *testing.T) {
	t.Run("nil callback", func(t *testing.T) {
		opts := ControllerOptions{}
		if name := opts.CountryName("US"); name != "US" {
			t.Errorf("expected US, got %s", name)
		}
	})

	t.Run("empty code", func(t *testing.T) {
		opts := ControllerOptions{
			CountryNameByIso2: func(iso2 string) (string, error) {
				return "United States", nil
			},
		}
		if name := opts.CountryName(""); name != "" {
			t.Errorf("expected empty string, got %s", name)
		}
	})

	t.Run("valid lookup", func(t *testing.T) {
		opts := ControllerOptions{
			CountryNameByIso2: func(iso2 string) (string, error) {
				if iso2 == "US" {
					return "United States", nil
				}
				return "", nil
			},
		}
		if name := opts.CountryName("US"); name != "United States" {
			t.Errorf("expected United States, got %s", name)
		}
	})

	t.Run("unknown code returning empty string", func(t *testing.T) {
		opts := ControllerOptions{
			CountryNameByIso2: func(iso2 string) (string, error) {
				return "", nil
			},
		}
		if name := opts.CountryName("XX"); name != "XX" {
			t.Errorf("expected raw code XX when callback returns empty string, got %s", name)
		}
	})
}

// mockVisitor implements VisitorLike for testing.
type mockVisitor struct {
	path        string
	country     string
	browser     string
	os          string
	device      string
	deviceType  string
	ip          string
	fingerprint string
	createdAt   string
	bot         string
	threat      string
}

func (m mockVisitor) GetPath() string           { return m.path }
func (m mockVisitor) GetCountry() string        { return m.country }
func (m mockVisitor) GetUserBrowser() string    { return m.browser }
func (m mockVisitor) GetUserOs() string         { return m.os }
func (m mockVisitor) GetUserDevice() string     { return m.device }
func (m mockVisitor) GetUserDeviceType() string { return m.deviceType }
func (m mockVisitor) GetIpAddress() string      { return m.ip }
func (m mockVisitor) GetFingerprint() string    { return m.fingerprint }
func (m mockVisitor) GetCreatedAt() string      { return m.createdAt }
func (m mockVisitor) GetBot() string            { return m.bot }
func (m mockVisitor) GetThreat() string         { return m.threat }

func TestIsBotVisitor(t *testing.T) {
	if IsBotVisitor(nil) {
		t.Error("nil visitor should not be bot")
	}
	if !IsBotVisitor(mockVisitor{bot: "yes"}) {
		t.Error("bot=yes should be bot")
	}
	if !IsBotVisitor(mockVisitor{bot: "no", deviceType: "bot"}) {
		t.Error("deviceType=bot should be bot even if bot=no")
	}
	if !IsBotVisitor(mockVisitor{bot: "no", device: "bot"}) {
		t.Error("device=bot should be bot even if bot=no")
	}
	if IsBotVisitor(mockVisitor{bot: "no", deviceType: "desktop"}) {
		t.Error("desktop with bot=no should not be bot")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
