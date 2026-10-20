package settings

import (
	"net/http"

	"github.com/dracory/req"
	"github.com/dracory/statsadmin/shared"
)

type settingsController struct {
	opts shared.ControllerOptions
}

func NewSettingsController(opts shared.ControllerOptions) *settingsController {
	return &settingsController{opts: opts}
}

func (controller *settingsController) Handler(w http.ResponseWriter, r *http.Request) string {
	action := req.GetStringTrimmed(r, "action")

	switch action {
	case actionLoadIPs:
		return controller.handleLoadIPs(w, r)
	case actionAddIP:
		return controller.handleAddIP(w, r)
	case actionRemoveIP:
		return controller.handleRemoveIP(w, r)
	case actionDeleteVisitors:
		return controller.handleDeleteVisitors(w, r)
	case actionLoadBots:
		return controller.handleLoadBots(w, r)
	case actionAddBot:
		return controller.handleAddBot(w, r)
	case actionRemoveBot:
		return controller.handleRemoveBot(w, r)
	case actionIdentifyBots:
		return controller.handleIdentifyBots(w, r)
	case actionDeleteBots:
		return controller.handleDeleteBots(w, r)
	case actionDeleteThreats:
		return controller.handleDeleteThreats(w, r)
	case actionLoadStats:
		return controller.handleLoadStats(w, r)
	default:
		return controller.renderPage(w, r)
	}
}
