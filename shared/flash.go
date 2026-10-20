package shared

import "net/http"

// FlashOrRedirect renders a flash error message via the FlashError callback
// when configured, otherwise falls back to a plain HTTP redirect. It returns
// the response body string (matching the controller Handler signature).
func FlashOrRedirect(opts ControllerOptions, w http.ResponseWriter, r *http.Request, message string, redirectURL string, delaySeconds int) string {
	if opts.FlashError != nil {
		return opts.FlashError(w, r, message, redirectURL, delaySeconds)
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return ""
}
