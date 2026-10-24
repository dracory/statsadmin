package statsadmin

import (
	"net/http"
	"strings"

	"github.com/dracory/api"
	"github.com/dracory/req"
	"github.com/dracory/statsadmin/dashboard"
	"github.com/dracory/statsadmin/ipdetails"
	"github.com/dracory/statsadmin/sessions"
	"github.com/dracory/statsadmin/settings"
	"github.com/dracory/statsadmin/shared"
	"github.com/dracory/statsadmin/visitors"
)

// Handler dispatches the request to the matching controller based on the
// "controller" query parameter. It writes the response directly to the
// ResponseWriter.
func Handler(opts shared.ControllerOptions, w http.ResponseWriter, r *http.Request) {
	controller := req.GetStringTrimmed(r, shared.ParamController)
	action := req.GetStringTrimmed(r, shared.ParamAction)

	// Baseline security headers for every admin response (pages, AJAX, CSV).
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")

	// Action/AJAX endpoints are gated centrally: they read PII and mutate or
	// export data, so every current and future action requires authentication
	// (when AuthUserID is configured), a POST method, and the X-Requested-With
	// header as a CSRF countermeasure (a cross-origin form or simple fetch
	// cannot set a custom header without a CORS preflight).
	// Page renders (empty action) are exempt — each controller's renderPage
	// enforces its own auth check and redirect.
	if action != "" {
		if opts.AuthUserID != nil && opts.AuthUserID(r) == "" {
			api.RespondWithStatusCode(w, r, api.Unauthenticated("Authentication required"), http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost ||
			!strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
			api.RespondWithStatusCode(w, r, api.Forbidden("Action endpoints require POST with X-Requested-With: XMLHttpRequest"), http.StatusForbidden)
			return
		}
	}

	var body string
	switch controller {
	case shared.CONTROLLER_VISITORS:
		body = visitors.NewVisitorsController(opts).Handler(w, r)
	case shared.CONTROLLER_SESSIONS:
		body = sessions.NewSessionsController(opts).Handler(w, r)
	case shared.CONTROLLER_SETTINGS:
		body = settings.NewSettingsController(opts).Handler(w, r)
	case shared.CONTROLLER_IP_DETAILS:
		body = ipdetails.NewIPDetailsController(opts).Handler(w, r)
	default:
		body = dashboard.NewDashboardController(opts).Handler(w, r)
	}

	// Controllers return an HTML string for page renders; AJAX endpoints
	// write directly to the ResponseWriter and return "".
	if body != "" {
		// #nosec G705 -- body is server-rendered HTML produced by the page
		// controllers (built with hb, which escapes interpolated values),
		// not raw user input.
		_, _ = w.Write([]byte(body))
	}
}
