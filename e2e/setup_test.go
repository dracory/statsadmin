// Package e2e contains end-to-end tests that drive the statsadmin panel
// with a real headless Chrome via chromedp (CDP). The admin is served
// in-process by httptest on a random port — no binary build needed.
package e2e

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/dracory/statsadmin"
	"github.com/dracory/statsstore"
	_ "modernc.org/sqlite"
)

var (
	baseURL   string
	allocCtx  context.Context
	testStore statsstore.StoreInterface

	seedIP      = "203.0.113.10"
	seedIP2     = "198.51.100.7"
	seedBotIP   = "66.249.66.1"
	seedPath    = "/e2e-seed-page"
	seedCountry = "US"
)

const adminPath = "/admin/stats"

func TestMain(m *testing.M) {
	// In-process admin server backed by a seeded in-memory SQLite store.
	db, err := sql.Open("sqlite", ":memory:?parseTime=true")
	if err != nil {
		panic("e2e: failed to open db: " + err.Error())
	}

	store, err := statsstore.NewStore(statsstore.NewStoreOptions{
		DB:                 db,
		VisitorTableName:   "visitors_e2e",
		AutomigrateEnabled: true,
	})
	if err != nil {
		panic("e2e: failed to create store: " + err.Error())
	}
	testStore = store

	seedStore(store)

	panel, err := statsadmin.New(statsadmin.AdminOptions{
		Store:   store,
		Logger:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
		HomeURL: "/admin",
		BaseURL: adminPath,
	})
	if err != nil {
		panic("e2e: failed to create statsadmin: " + err.Error())
	}

	mux := http.NewServeMux()
	mux.Handle(adminPath, http.HandlerFunc(panel.Handle))
	mux.Handle(adminPath+"/", http.HandlerFunc(panel.Handle))
	srv := httptest.NewServer(mux)
	baseURL = srv.URL

	// One shared browser process for the entire test run.
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Headless)
	if bin := chromeBin(); bin != "" {
		opts = append(opts, chromedp.ExecPath(bin))
	}
	var cancel context.CancelFunc
	allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), opts...)

	code := m.Run()

	cancel()
	srv.Close()
	os.Exit(code)
}

// seedStore populates the store with visitor records so the dashboard,
// visitors and sessions pages have rows to render.
func seedStore(store statsstore.StoreInterface) {
	ctx := context.Background()

	seed := func(ip, path, country, ua string) {
		v := statsstore.NewVisitor().
			SetIpAddress(ip).
			SetPath(path).
			SetCountry(country).
			SetUserAgent(ua).
			SetBot(statsstore.VALUE_NO)
		if err := store.VisitorCreate(ctx, v); err != nil {
			panic("e2e: failed to seed visitor: " + err.Error())
		}
	}

	// Two visits from the same IP — one session with two page views.
	seed(seedIP, seedPath, seedCountry,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0")
	seed(seedIP, "/about", seedCountry,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0")

	// A second IP — second session.
	seed(seedIP2, "/blog/post-1", "DE",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15")

	// A self-identifying crawler (bot flag left "no" so it exercises the
	// identify-bots candidate path too).
	seed(seedBotIP, "/robots.txt", seedCountry,
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
}

// newTab opens a fresh tab in the shared browser and returns its
// context. Each tab gets a 30s deadline; cleanup closes the tab.
func newTab(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := chromedp.NewContext(allocCtx)
	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() { cancelTimeout(); cancel() })
	return ctx
}

// chromeBin resolves a headless Chromium binary. Order: CHROME_BIN env,
// Playwright's downloaded headless shell, empty (chromedp finds Chrome
// on PATH / standard install locations).
func chromeBin() string {
	if b := os.Getenv("CHROME_BIN"); b != "" {
		return b
	}
	patterns := []string{
		// Windows
		filepath.Join(os.Getenv("LOCALAPPDATA"),
			"ms-playwright/chromium_headless_shell-*/chrome-headless-shell-win64/chrome-headless-shell.exe"),
		// WSL: Windows-side Playwright shell via interop
		"/mnt/*/Users/*/AppData/Local/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-win64/chrome-headless-shell.exe",
		// Linux
		filepath.Join(os.Getenv("HOME"),
			".cache/ms-playwright/chromium_headless_shell-*/chrome-headless-shell-linux64/chrome-headless-shell"),
	}
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[len(matches)-1]
		}
	}
	return ""
}
