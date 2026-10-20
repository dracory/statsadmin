package dashboard

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/dracory/cdn"
	"github.com/dracory/hb"
	"github.com/dracory/statsadmin/shared"
)

var (
	//go:embed dashboard.html
	dashboardHTML string

	//go:embed dashboard.js
	dashboardJS string
)

func (controller *dashboardController) renderPage(w http.ResponseWriter, r *http.Request) string {
	// Auth check: redirect when unauthenticated (when callback is configured).
	if controller.opts.AuthUserID != nil && controller.opts.AuthUserID(r) == "" {
		return shared.FlashOrRedirect(controller.opts, w, r, "You are not logged in. Please login to continue.", controller.opts.HomeURL, 10)
	}

	if controller.opts.Store == nil {
		return shared.FlashOrRedirect(controller.opts, w, r, "Stats store not available", controller.opts.HomeURL, 10)
	}

	breadcrumbs := shared.Breadcrumbs([]shared.Breadcrumb{
		{Name: "Home", URL: controller.opts.HomeURL},
		{Name: "Stats", URL: controller.opts.BaseURL},
		{Name: "Dashboard", URL: ""},
	})

	heading := hb.Heading1().HTML("Visitor Analytics")

	linksHelper := shared.NewLinks(controller.opts.BaseURL)
	urlLoadDashboard := linksHelper.Dashboard(map[string]string{"action": actionLoadDashboard})
	urlVisitors := linksHelper.Visitors(nil)
	urlSettings := linksHelper.Settings(nil)
	urlDashboard := linksHelper.Dashboard(nil)
	urlIPDetailsBase := linksHelper.IPDetails(nil)
	urlVisitorsBase := linksHelper.Visitors(nil)
	urlSessions := linksHelper.Sessions(nil)

	html := strings.ReplaceAll(dashboardHTML, "urlLoadDashboard", "'"+urlLoadDashboard+"'")
	html = strings.ReplaceAll(html, "urlIPDetailsBase", "'"+urlIPDetailsBase+"'")
	html = strings.ReplaceAll(html, "urlVisitorsBase", "'"+urlVisitorsBase+"'")
	html = strings.ReplaceAll(html, "__URL_DASHBOARD__", urlDashboard)
	html = strings.ReplaceAll(html, "__URL_VISITORS__", urlVisitors)
	html = strings.ReplaceAll(html, "__URL_SESSIONS__", urlSessions)
	html = strings.ReplaceAll(html, "__URL_SETTINGS__", urlSettings)
	js := strings.ReplaceAll(dashboardJS, "urlLoadDashboard", "'"+urlLoadDashboard+"'")
	js = strings.ReplaceAll(js, "urlIPDetailsBase", "'"+urlIPDetailsBase+"'")
	js = strings.ReplaceAll(js, "urlVisitorsBase", "'"+urlVisitorsBase+"'")

	vueCDN := hb.Script("").Src(cdn.VueJs_3_5_32())

	content := hb.Div().
		Class("container").
		Child(heading).
		Child(breadcrumbs).
		Child(hb.HR()).
		Child(vueCDN).
		Child(hb.Raw(html)).
		Child(hb.Script(js))

	// Render via the injected LayoutInterface.
	controller.opts.Layout.SetTitle("Visitor Analytics | Stats")
	controller.opts.Layout.SetBody(content.ToHTML())
	controller.opts.Layout.SetScriptURLs([]string{cdn.Notiflix_3_2_8()})
	controller.opts.Layout.SetScripts([]string{})
	controller.opts.Layout.SetStyles([]string{cdn.Notiflix_3_2_8_CSS()})

	return controller.opts.Layout.Render(w, r)
}
