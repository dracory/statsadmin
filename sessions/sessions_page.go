package sessions

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/dracory/cdn"
	"github.com/dracory/hb"
	"github.com/dracory/statsadmin/shared"
)

var (
	//go:embed sessions.html
	sessionsHTML string

	//go:embed sessions.js
	sessionsJS string
)

func (controller *sessionsController) renderPage(w http.ResponseWriter, r *http.Request) string {
	if controller.opts.AuthUserID != nil && controller.opts.AuthUserID(r) == "" {
		return shared.FlashOrRedirect(controller.opts, w, r, "You are not logged in. Please login to continue.", controller.opts.HomeURL, 10)
	}

	if controller.opts.Store == nil {
		return shared.FlashOrRedirect(controller.opts, w, r, "Stats store not available", controller.opts.HomeURL, 10)
	}

	breadcrumbs := shared.Breadcrumbs([]shared.Breadcrumb{
		{Name: "Home", URL: controller.opts.HomeURL},
		{Name: "Stats", URL: controller.opts.BaseURL},
		{Name: "Sessions", URL: ""},
	})

	heading := hb.Heading1().HTML("Sessions")

	linksHelper := shared.NewLinks(controller.opts.BaseURL)
	urlLoadSessions := linksHelper.Sessions(map[string]string{"action": actionLoadSessions})
	urlSessions := linksHelper.Sessions(nil)
	urlDashboard := linksHelper.Dashboard(nil)
	urlVisitors := linksHelper.Visitors(nil)
	urlSettings := linksHelper.Settings(nil)
	urlIPDetailsBase := linksHelper.IPDetails(nil)

	html := strings.ReplaceAll(sessionsHTML, "urlLoadSessions", "'"+urlLoadSessions+"'")
	html = strings.ReplaceAll(html, "urlIPDetailsBase", "'"+urlIPDetailsBase+"'")
	html = strings.ReplaceAll(html, "__URL_DASHBOARD__", urlDashboard)
	html = strings.ReplaceAll(html, "__URL_VISITORS__", urlVisitors)
	html = strings.ReplaceAll(html, "__URL_SESSIONS__", urlSessions)
	html = strings.ReplaceAll(html, "__URL_SETTINGS__", urlSettings)
	js := strings.ReplaceAll(sessionsJS, "urlLoadSessions", "'"+urlLoadSessions+"'")
	js = strings.ReplaceAll(js, "urlIPDetailsBase", "'"+urlIPDetailsBase+"'")

	vueCDN := hb.Script("").Src(cdn.VueJs_3_5_32())

	content := hb.Div().
		Class("container").
		Child(heading).
		Child(breadcrumbs).
		Child(hb.HR()).
		Child(vueCDN).
		Child(hb.Raw(html)).
		Child(hb.Script(js))

	controller.opts.Layout.SetTitle("Sessions | Stats")
	controller.opts.Layout.SetBody(content.ToHTML())
	controller.opts.Layout.SetScriptURLs([]string{cdn.Notiflix_3_2_8()})
	controller.opts.Layout.SetScripts([]string{})
	controller.opts.Layout.SetStyleURLs([]string{shared.FlagIconsCSS})
	controller.opts.Layout.SetStyles([]string{cdn.Notiflix_3_2_8_CSS()})

	return controller.opts.Layout.Render(w, r)
}
