package settings

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/dracory/cdn"
	"github.com/dracory/hb"
	"github.com/dracory/statsadmin/shared"
)

var (
	//go:embed settings.html
	settingsHTML string

	//go:embed settings.js
	settingsJS string
)

func (controller *settingsController) renderPage(w http.ResponseWriter, r *http.Request) string {
	if controller.opts.AuthUserID != nil && controller.opts.AuthUserID(r) == "" {
		return shared.FlashOrRedirect(controller.opts, w, r, "You are not logged in. Please login to continue.", controller.opts.HomeURL, 10)
	}

	if controller.opts.Store == nil {
		return shared.FlashOrRedirect(controller.opts, w, r, "Stats store not available", controller.opts.HomeURL, 10)
	}

	breadcrumbs := shared.Breadcrumbs([]shared.Breadcrumb{
		{Name: "Home", URL: controller.opts.HomeURL},
		{Name: "Stats", URL: controller.opts.BaseURL},
		{Name: "Settings", URL: ""},
	})

	heading := hb.Heading1().HTML("Stats Settings")

	linksHelper := shared.NewLinks(controller.opts.BaseURL)
	urlLoadIPs := linksHelper.Settings(map[string]string{"action": actionLoadIPs})
	urlAddIP := linksHelper.Settings(map[string]string{"action": actionAddIP})
	urlRemoveIP := linksHelper.Settings(map[string]string{"action": actionRemoveIP})
	urlDeleteVisitors := linksHelper.Settings(map[string]string{"action": actionDeleteVisitors})
	urlLoadExcludedPaths := linksHelper.Settings(map[string]string{"action": actionLoadExcludedPaths})
	urlAddExcludedPath := linksHelper.Settings(map[string]string{"action": actionAddExcludedPath})
	urlRemoveExcludedPath := linksHelper.Settings(map[string]string{"action": actionRemoveExcludedPath})
	urlLoadIncludedPaths := linksHelper.Settings(map[string]string{"action": actionLoadIncludedPaths})
	urlAddIncludedPath := linksHelper.Settings(map[string]string{"action": actionAddIncludedPath})
	urlRemoveIncludedPath := linksHelper.Settings(map[string]string{"action": actionRemoveIncludedPath})
	urlLoadBots := linksHelper.Settings(map[string]string{"action": actionLoadBots})
	urlAddBot := linksHelper.Settings(map[string]string{"action": actionAddBot})
	urlRemoveBot := linksHelper.Settings(map[string]string{"action": actionRemoveBot})
	urlIdentifyBots := linksHelper.Settings(map[string]string{"action": actionIdentifyBots})
	urlDeleteBots := linksHelper.Settings(map[string]string{"action": actionDeleteBots})
	urlDeleteThreats := linksHelper.Settings(map[string]string{"action": actionDeleteThreats})
	urlLoadStats := linksHelper.Settings(map[string]string{"action": actionLoadStats})
	urlIPDetailsBase := linksHelper.IPDetails(nil)
	urlDashboard := linksHelper.Dashboard(nil)
	urlVisitors := linksHelper.Visitors(nil)
	urlSessions := linksHelper.Sessions(nil)
	urlSettings := linksHelper.Settings(nil)

	html := strings.ReplaceAll(settingsHTML, "urlLoadIPs", "'"+urlLoadIPs+"'")
	html = strings.ReplaceAll(html, "urlAddIP", "'"+urlAddIP+"'")
	html = strings.ReplaceAll(html, "urlRemoveIP", "'"+urlRemoveIP+"'")
	html = strings.ReplaceAll(html, "urlDeleteVisitors", "'"+urlDeleteVisitors+"'")
	html = strings.ReplaceAll(html, "urlLoadExcludedPaths", "'"+urlLoadExcludedPaths+"'")
	html = strings.ReplaceAll(html, "urlAddExcludedPath", "'"+urlAddExcludedPath+"'")
	html = strings.ReplaceAll(html, "urlRemoveExcludedPath", "'"+urlRemoveExcludedPath+"'")
	html = strings.ReplaceAll(html, "urlLoadIncludedPaths", "'"+urlLoadIncludedPaths+"'")
	html = strings.ReplaceAll(html, "urlAddIncludedPath", "'"+urlAddIncludedPath+"'")
	html = strings.ReplaceAll(html, "urlRemoveIncludedPath", "'"+urlRemoveIncludedPath+"'")
	html = strings.ReplaceAll(html, "urlLoadBots", "'"+urlLoadBots+"'")
	html = strings.ReplaceAll(html, "urlAddBot", "'"+urlAddBot+"'")
	html = strings.ReplaceAll(html, "urlRemoveBot", "'"+urlRemoveBot+"'")
	html = strings.ReplaceAll(html, "urlIdentifyBots", "'"+urlIdentifyBots+"'")
	html = strings.ReplaceAll(html, "urlDeleteBots", "'"+urlDeleteBots+"'")
	html = strings.ReplaceAll(html, "urlDeleteThreats", "'"+urlDeleteThreats+"'")
	html = strings.ReplaceAll(html, "urlLoadStats", "'"+urlLoadStats+"'")
	html = strings.ReplaceAll(html, "urlIPDetailsBase", "'"+urlIPDetailsBase+"'")
	html = strings.ReplaceAll(html, "__URL_DASHBOARD__", urlDashboard)
	html = strings.ReplaceAll(html, "__URL_VISITORS__", urlVisitors)
	html = strings.ReplaceAll(html, "__URL_SESSIONS__", urlSessions)
	html = strings.ReplaceAll(html, "__URL_SETTINGS__", urlSettings)
	js := strings.ReplaceAll(settingsJS, "urlLoadIPs", "'"+urlLoadIPs+"'")
	js = strings.ReplaceAll(js, "urlAddIP", "'"+urlAddIP+"'")
	js = strings.ReplaceAll(js, "urlRemoveIP", "'"+urlRemoveIP+"'")
	js = strings.ReplaceAll(js, "urlDeleteVisitors", "'"+urlDeleteVisitors+"'")
	js = strings.ReplaceAll(js, "urlLoadExcludedPaths", "'"+urlLoadExcludedPaths+"'")
	js = strings.ReplaceAll(js, "urlAddExcludedPath", "'"+urlAddExcludedPath+"'")
	js = strings.ReplaceAll(js, "urlRemoveExcludedPath", "'"+urlRemoveExcludedPath+"'")
	js = strings.ReplaceAll(js, "urlLoadIncludedPaths", "'"+urlLoadIncludedPaths+"'")
	js = strings.ReplaceAll(js, "urlAddIncludedPath", "'"+urlAddIncludedPath+"'")
	js = strings.ReplaceAll(js, "urlRemoveIncludedPath", "'"+urlRemoveIncludedPath+"'")
	js = strings.ReplaceAll(js, "urlLoadBots", "'"+urlLoadBots+"'")
	js = strings.ReplaceAll(js, "urlAddBot", "'"+urlAddBot+"'")
	js = strings.ReplaceAll(js, "urlRemoveBot", "'"+urlRemoveBot+"'")
	js = strings.ReplaceAll(js, "urlIdentifyBots", "'"+urlIdentifyBots+"'")
	js = strings.ReplaceAll(js, "urlDeleteBots", "'"+urlDeleteBots+"'")
	js = strings.ReplaceAll(js, "urlDeleteThreats", "'"+urlDeleteThreats+"'")
	js = strings.ReplaceAll(js, "urlLoadStats", "'"+urlLoadStats+"'")
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

	controller.opts.Layout.SetTitle("Settings | Stats")
	controller.opts.Layout.SetBody(content.ToHTML())
	controller.opts.Layout.SetScriptURLs([]string{cdn.Notiflix_3_2_8()})
	controller.opts.Layout.SetScripts([]string{})
	controller.opts.Layout.SetStyleURLs([]string{shared.FlagIconsCSS})
	controller.opts.Layout.SetStyles([]string{cdn.Notiflix_3_2_8_CSS()})

	return controller.opts.Layout.Render(w, r)
}
