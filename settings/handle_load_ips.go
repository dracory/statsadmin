package settings

import (
	"net/http"

	"github.com/dracory/api"
)

func (controller *settingsController) handleLoadIPs(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	ips, err := store.ExcludedIPList(r.Context())
	if err != nil {
		api.Respond(w, r, api.Error(err.Error()))
		return ""
	}

	ipList := make([]map[string]any, 0, len(ips))
	for _, ip := range ips {
		ipList = append(ipList, map[string]any{
			FieldIP: ip,
		})
	}

	api.Respond(w, r, api.SuccessWithData("Excluded IPs loaded", map[string]any{
		FieldIPs:   ipList,
		FieldTotal: len(ips),
	}))
	return ""
}
