package statsadmin

import (
	"net/http"

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
		_, _ = w.Write([]byte(body))
	}
}
