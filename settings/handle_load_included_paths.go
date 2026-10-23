package settings

import (
	"net/http"

	"github.com/dracory/api"
)

func (controller *settingsController) handleLoadIncludedPaths(w http.ResponseWriter, r *http.Request) string {
	if r.Method != http.MethodPost {
		api.Respond(w, r, api.Error("Method not allowed"))
		return ""
	}

	store := controller.opts.Store
	if store == nil {
		api.Respond(w, r, api.Error("Stats store not available"))
		return ""
	}

	paths, err := store.IncludedPathList(r.Context())
	if err != nil {
		api.Respond(w, r, api.Error("Failed to load included paths: "+err.Error()))
		return ""
	}

	pathList := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		pathList = append(pathList, map[string]any{
			FieldPath: path,
		})
	}

	api.Respond(w, r, api.SuccessWithData("Included paths loaded", map[string]any{
		FieldPaths: pathList,
		FieldTotal: len(paths),
	}))
	return ""
}
