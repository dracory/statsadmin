package shared

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/dracory/statsstore"
	_ "modernc.org/sqlite"
)

// NewTestStore creates an in-memory SQLite statsstore for testing.
// It auto-migrates the visitor table and registers a cleanup via t.Cleanup.
func NewTestStore(t testing.TB) statsstore.StoreInterface {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?parseTime=true")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	store, err := statsstore.NewStore(statsstore.NewStoreOptions{
		DB:                 db,
		VisitorTableName:   "visitors_test",
		AutomigrateEnabled: true,
	})
	if err != nil {
		_ = db.Close()
		t.Fatalf("failed to create store: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})
	return store
}

// FakeLayout implements LayoutInterface for testing. It captures all Set*
// calls and returns a fixed string from Render.
type FakeLayout struct {
	Title      string
	Body       string
	Scripts    []string
	ScriptURLs []string
	Styles     []string
	StyleURLs  []string
}

func (l *FakeLayout) SetTitle(title string)                       { l.Title = title }
func (l *FakeLayout) SetScriptURLs(urls []string)                 { l.ScriptURLs = append([]string{}, urls...) }
func (l *FakeLayout) SetScripts(scripts []string)                 { l.Scripts = append([]string{}, scripts...) }
func (l *FakeLayout) SetStyleURLs(styles []string)                { l.StyleURLs = append([]string{}, styles...) }
func (l *FakeLayout) SetStyles(styles []string)                   { l.Styles = append([]string{}, styles...) }
func (l *FakeLayout) SetBody(body string)                         { l.Body = body }
func (l *FakeLayout) SetCountryNameByIso2(func(string) (string, error)) {}
func (l *FakeLayout) Render(http.ResponseWriter, *http.Request) string { return l.Body }

// NewTestControllerOptions builds a ControllerOptions wired to a test store
// and FakeLayout, suitable for controller tests.
func NewTestControllerOptions(t testing.TB) (ControllerOptions, *FakeLayout, statsstore.StoreInterface) {
	t.Helper()
	store := NewTestStore(t)
	layout := &FakeLayout{}
	opts := ControllerOptions{
		Store:   store,
		Layout:  layout,
		HomeURL: "/admin",
		BaseURL: "/admin/stats",
		Logger:  slog.Default(),
	}
	return opts, layout, store
}

// SeedVisitor creates a single visitor record in the store for testing.
func SeedVisitor(t testing.TB, store statsstore.StoreInterface, ip, path, country string) {
	t.Helper()
	_ = SeedVisitorWithBot(t, store, ip, path, country, statsstore.VALUE_NO)
}

// SeedVisitorWithBot creates a visitor record with a specified bot flag.
func SeedVisitorWithBot(t testing.TB, store statsstore.StoreInterface, ip, path, country, bot string) statsstore.VisitorInterface {
	t.Helper()
	v := statsstore.NewVisitor().
		SetIpAddress(ip).
		SetPath(path).
		SetCountry(country).
		SetBot(bot)
	if err := store.VisitorCreate(context.Background(), v); err != nil {
		t.Fatalf("failed to seed visitor: %v", err)
	}
	return v
}

// AssertContains checks that the body contains a substring, failing the test
// with a clear message if not.
func AssertContains(t testing.TB, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("expected body to contain %q, got: %s", substr, body)
	}
}
