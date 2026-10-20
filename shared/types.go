package shared

import (
	"log/slog"
	"net/http"

	"github.com/dracory/statsstore"
)

// LayoutInterface defines the layout methods needed by controllers.
// Consumers implement this to wrap admin pages in their own chrome.
type LayoutInterface interface {
	SetTitle(title string)
	SetScriptURLs(scripts []string)
	SetScripts(scripts []string)
	SetStyleURLs(styles []string)
	SetStyles(styles []string)
	SetBody(string)
	// SetCountryNameByIso2 provides country lookup helpers used by the admin UI.
	SetCountryNameByIso2(func(iso2Code string) (string, error))
	Render(w http.ResponseWriter, r *http.Request) string
}

// Breadcrumb represents a navigation breadcrumb entry.
type Breadcrumb struct {
	Name string
	URL  string
}

// ControllerOptions contains the dependencies and configuration for creating
// an admin controller. It extends the original statsstore admin options with
// optional callbacks for auth, flash messages, and base URL configuration so
// the package remains self-contained and reusable by any project.
type ControllerOptions struct {
	// Store is the statsstore instance used for all visitor queries.
	Store statsstore.StoreInterface

	// Layout renders the full HTML page. Required.
	Layout LayoutInterface

	// HomeURL is the URL for the admin home page (e.g. "/admin").
	HomeURL string

	// WebsiteUrl is the public site URL.
	WebsiteUrl string

	// BaseURL is the base URL for the stats admin (e.g. "/admin/stats").
	// Replaces the old internal/links dependency.
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

// CountryName resolves an ISO2 code to a human-readable country name using
// the CountryNameByIso2 callback. Returns the raw code when the callback is
// nil or returns an error.
func (o ControllerOptions) CountryName(iso2 string) string {
	if iso2 == "" {
		return ""
	}
	if o.CountryNameByIso2 == nil {
		return iso2
	}
	name, err := o.CountryNameByIso2(iso2)
	if err != nil || name == "" {
		return iso2
	}
	return name
}
