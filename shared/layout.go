package shared

import (
	"net/http"

	"github.com/dracory/cdn"
	"github.com/dracory/hb"
)

// FuncLayoutAdapter adapts a shopadmin/blogadmin-style FuncLayout callback to
// the LayoutInterface used by the admin controllers. Page-level options set via
// the Set* methods are accumulated and passed to FuncLayout at Render time.
type FuncLayoutAdapter struct {
	// FuncLayout renders the final page. Required for Render to produce output.
	FuncLayout func(w http.ResponseWriter, r *http.Request, title string, body string, options struct {
		Styles     []string
		StyleURLs  []string
		Scripts    []string
		ScriptURLs []string
	}) string

	title      string
	body       string
	styles     []string
	styleURLs  []string
	scripts    []string
	scriptURLs []string
}

// SetTitle sets the page title.
func (a *FuncLayoutAdapter) SetTitle(title string) { a.title = title }

// SetScriptURLs sets extra script URLs.
func (a *FuncLayoutAdapter) SetScriptURLs(urls []string) { a.scriptURLs = urls }

// SetScripts sets extra inline scripts.
func (a *FuncLayoutAdapter) SetScripts(scripts []string) { a.scripts = scripts }

// SetStyleURLs sets extra stylesheet URLs.
func (a *FuncLayoutAdapter) SetStyleURLs(urls []string) { a.styleURLs = urls }

// SetStyles sets extra inline styles.
func (a *FuncLayoutAdapter) SetStyles(styles []string) { a.styles = styles }

// SetBody sets the page body HTML.
func (a *FuncLayoutAdapter) SetBody(body string) { a.body = body }

// SetCountryNameByIso2 is a no-op for the adapter; country name resolution is
// provided through ControllerOptions.CountryNameByIso2 instead.
func (a *FuncLayoutAdapter) SetCountryNameByIso2(func(iso2Code string) (string, error)) {}

// Render renders the page through the configured FuncLayout.
func (a *FuncLayoutAdapter) Render(w http.ResponseWriter, r *http.Request) string {
	if a.FuncLayout == nil {
		return a.body
	}
	return a.FuncLayout(w, r, a.title, a.body, struct {
		Styles     []string
		StyleURLs  []string
		Scripts    []string
		ScriptURLs []string
	}{
		Styles:     a.styles,
		StyleURLs:  a.styleURLs,
		Scripts:    a.scripts,
		ScriptURLs: a.scriptURLs,
	})
}

// Layout is the default layout renderer used when the consumer does not
// provide AdminOptions.FuncLayout. It builds a complete HTML page with
// Bootstrap + Vue + SweetAlert2 via CDN, matching the shopadmin/blogadmin
// default layout convention.
func Layout(w http.ResponseWriter, r *http.Request, title, body string, options struct {
	Styles     []string
	StyleURLs  []string
	Scripts    []string
	ScriptURLs []string
}) string {
	_ = r
	_ = w
	return webpageComplete(title, body, options).ToHTML()
}

// webpageComplete builds the default webpage template.
func webpageComplete(title, content string, options struct {
	Styles     []string
	StyleURLs  []string
	Scripts    []string
	ScriptURLs []string
}) *hb.HtmlWebpage {
	webpage := hb.NewWebpage()
	webpage.SetTitle(title)
	webpage.AddStyleURLs([]string{
		cdn.BootstrapCss_5_3_3(),
		"https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css",
	})
	webpage.AddStyleURLs(options.StyleURLs)
	webpage.AddScriptURLs([]string{
		cdn.BootstrapJs_5_3_3(),
		cdn.Jquery_3_7_1(),
		cdn.VueJs_3(),
		cdn.Sweetalert2_11(),
	})
	webpage.AddScriptURLs(options.ScriptURLs)
	webpage.AddScripts(options.Scripts)
	webpage.AddStyles(options.Styles)
	webpage.AddChild(hb.NewHTML(content))
	return webpage
}
