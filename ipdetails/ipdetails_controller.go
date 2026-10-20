package ipdetails

import (
	"net/http"

	"github.com/dracory/req"
	"github.com/dracory/statsadmin/shared"
)

type ipDetailsController struct {
	opts shared.ControllerOptions
}

func NewIPDetailsController(opts shared.ControllerOptions) *ipDetailsController {
	return &ipDetailsController{opts: opts}
}

func (controller *ipDetailsController) Handler(w http.ResponseWriter, r *http.Request) string {
	action := req.GetStringTrimmed(r, "action")

	switch action {
	case actionLoadPaths:
		return controller.handleLoadPaths(w, r)
	case actionLoadBotReasons:
		return controller.handleLoadBotReasons(w, r)
	case actionFlagBot:
		return controller.handleFlagBot(w, r)
	case actionFlagThreat:
		return controller.handleFlagThreat(w, r)
	case actionRemoveEntries:
		return controller.handleRemoveEntries(w, r)
	default:
		return controller.renderPage(w, r)
	}
}
