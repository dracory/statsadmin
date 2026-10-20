package shared

import "net/url"

// Links builds stats admin URLs from a configured base URL.
// It mutates the passed params map by setting the controller key; callers
// should pass a fresh map each time.
type Links struct {
	baseURL string
}

// NewLinks returns a URL builder bound to the given base URL.
// If baseURL is empty, "/" is used as a safe default.
func NewLinks(baseURL string) *Links {
	if baseURL == "" {
		baseURL = "/"
	}
	return &Links{baseURL: baseURL}
}

// Base returns the configured base URL.
func (l *Links) Base() string {
	return l.baseURL
}

// buildURL appends query params to the base URL, setting the controller key.
func (l *Links) buildURL(controller string, params map[string]string) string {
	values := url.Values{}
	if params != nil {
		for k, v := range params {
			values.Set(k, v)
		}
	}
	values.Set(ParamController, controller)

	encoded := values.Encode()
	if encoded == "" {
		return l.baseURL
	}
	return l.baseURL + "?" + encoded
}

func (l *Links) Dashboard(params map[string]string) string {
	return l.buildURL(CONTROLLER_DASHBOARD, params)
}

func (l *Links) Visitors(params map[string]string) string {
	return l.buildURL(CONTROLLER_VISITORS, params)
}

func (l *Links) Sessions(params map[string]string) string {
	return l.buildURL(CONTROLLER_SESSIONS, params)
}

func (l *Links) Settings(params map[string]string) string {
	return l.buildURL(CONTROLLER_SETTINGS, params)
}

func (l *Links) IPDetails(params map[string]string) string {
	return l.buildURL(CONTROLLER_IP_DETAILS, params)
}
