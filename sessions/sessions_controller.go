package sessions

import (
	"net/http"

	"github.com/dracory/req"
	"github.com/dracory/statsadmin/shared"
)

type sessionsController struct {
	opts shared.ControllerOptions
}

// NewSessionsController creates a new sessions admin controller.
func NewSessionsController(opts shared.ControllerOptions) *sessionsController {
	return &sessionsController{opts: opts}
}

func (controller *sessionsController) Handler(w http.ResponseWriter, r *http.Request) string {
	action := req.GetStringTrimmed(r, "action")

	switch action {
	case actionLoadSessions:
		return controller.handleLoadSessions(w, r)
	case actionExportCSV:
		return controller.handleExportCSV(w, r)
	default:
		return controller.renderPage(w, r)
	}
}
