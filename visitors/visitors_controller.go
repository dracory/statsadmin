package visitors

import (
	"net/http"

	"github.com/dracory/req"
	"github.com/dracory/statsadmin/shared"
)

type visitorsController struct {
	opts shared.ControllerOptions
}

func NewVisitorsController(opts shared.ControllerOptions) *visitorsController {
	return &visitorsController{opts: opts}
}

func (controller *visitorsController) Handler(w http.ResponseWriter, r *http.Request) string {
	action := req.GetStringTrimmed(r, "action")

	switch action {
	case actionLoadVisitors:
		return controller.handleLoadVisitors(w, r)
	case actionExportCSV:
		return controller.handleExportCSV(w, r)
	default:
		return controller.renderPage(w, r)
	}
}
