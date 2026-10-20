// Package statsadmin provides a self-hosted visitor analytics admin dashboard.
//
// It is a self-contained, reusable library package: consumers inject a
// statsstore.StoreInterface and optional callbacks via
// statsadmin.New(statsadmin.AdminOptions{...}), and get the full dashboard
// for free. Rendering goes through FuncLayout so the host application can
// wrap the dashboard in its own chrome (branding, menus, theme).
//
// The package follows the folder-per-controller pattern: each controller
// lives in its own subfolder and renders via the shared layout.
package statsadmin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/dracory/statsadmin/shared"
	"github.com/dracory/statsstore"
)

// LayoutOptions is the anonymous struct passed to FuncLayout, matching the
// convention used by shopadmin/blogadmin so consumers can reuse the same
// layout function across dracory admin packages.
type LayoutOptions = struct {
	Styles     []string
	StyleURLs  []string
	Scripts    []string
	ScriptURLs []string
}

// AdminOptions contains all dependencies and configuration for the admin
// dashboard.
type AdminOptions struct {
	// Store is the statsstore instance used for all visitor queries. Required.
	Store statsstore.StoreInterface

	// FuncLayout renders an admin page inside the host application's layout
	// (branding, menus, theme). When nil, a default bare-bones HTML page is
	// used (Bootstrap + Vue CDN).
	FuncLayout func(w http.ResponseWriter, r *http.Request, title string, body string, options LayoutOptions) string

	// HomeURL is the URL for the admin home page (e.g. "/admin"). Required.
	HomeURL string

	// WebsiteUrl is the public site URL.
	WebsiteUrl string

	// BaseURL is the base URL for the stats admin (e.g. "/admin/stats").
	// Defaults to "/admin/stats" when empty.
	BaseURL string

	// CountryNameByIso2 maps an ISO2 code to a human-readable country name.
	// Optional; when nil, raw ISO2 codes are displayed.
	CountryNameByIso2 func(iso2Code string) (string, error)

	// AuthUserID returns the authenticated user ID from the request, or ""
	// when unauthenticated. When nil, auth checks are skipped.
	AuthUserID func(r *http.Request) string

	// FlashError renders a flash error message and returns the redirect
	// response body. When nil, controllers fall back to http.Redirect.
	FlashError func(w http.ResponseWriter, r *http.Request, message string, redirectURL string, delaySeconds int) string

	// Logger is used for internal logging. Defaults to slog.Default().
	Logger *slog.Logger
}

// AdminInterface defines the entry point for the stats admin. It satisfies
// http.Handler and also exposes the Handle method used by rtr-style routers.
type AdminInterface interface {
	http.Handler
	// Handle processes a stats admin request (alias for ServeHTTP).
	Handle(w http.ResponseWriter, r *http.Request)
}

// New creates a new stats admin instance. It validates required fields and
// returns an AdminInterface that dispatches to the appropriate controller
// based on the "controller" query parameter.
func New(opts AdminOptions) (AdminInterface, error) {
	if opts.Store == nil {
		return nil, errors.New("store is required")
	}
	if opts.HomeURL == "" {
		return nil, errors.New("home URL is required")
	}
	if opts.BaseURL == "" {
		opts.BaseURL = "/admin/stats"
	}
	if opts.FuncLayout == nil {
		opts.FuncLayout = shared.Layout
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	return &adminHandler{opts: opts}, nil
}

// adminHandler implements AdminInterface.
type adminHandler struct {
	opts AdminOptions
}

// Handle is an alias for ServeHTTP, for routers that use Handle-style handlers.
func (a *adminHandler) Handle(w http.ResponseWriter, r *http.Request) {
	a.ServeHTTP(w, r)
}

// ServeHTTP implements the http.Handler interface.
func (a *adminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	controllerOpts := shared.ControllerOptions{
		Store:             a.opts.Store,
		Layout:            &shared.FuncLayoutAdapter{FuncLayout: a.opts.FuncLayout},
		HomeURL:           a.opts.HomeURL,
		WebsiteUrl:        a.opts.WebsiteUrl,
		BaseURL:           a.opts.BaseURL,
		CountryNameByIso2: a.opts.CountryNameByIso2,
		AuthUserID:        a.opts.AuthUserID,
		FlashError:        a.opts.FlashError,
		Logger:            a.opts.Logger,
	}

	Handler(controllerOpts, w, r)
}
