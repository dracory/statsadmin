package dashboard

import (
	"net/http"

	"github.com/dracory/req"
	"github.com/dracory/statsadmin/shared"
)

// dashboardController handles the visitor analytics dashboard.
type dashboardController struct {
	opts shared.ControllerOptions
}

// NewDashboardController creates a new dashboard controller.
func NewDashboardController(opts shared.ControllerOptions) *dashboardController {
	return &dashboardController{opts: opts}
}

// Handler dispatches dashboard actions.
func (controller *dashboardController) Handler(w http.ResponseWriter, r *http.Request) string {
	action := req.GetStringTrimmed(r, "action")

	switch action {
	case actionLoadDashboard:
		return controller.handleLoadDashboard(w, r)
	case actionExportCSV:
		return controller.handleExportCSV(w, r)
	default:
		return controller.renderPage(w, r)
	}
}
