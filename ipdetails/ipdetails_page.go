package ipdetails

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/dracory/cdn"
	"github.com/dracory/hb"
	"github.com/dracory/req"
	"github.com/dracory/statsadmin/shared"
)

var (
	//go:embed ipdetails.html
	ipdetailsHTML string

	//go:embed ipdetails.js
	ipdetailsJS string
)

func (controller *ipDetailsController) renderPage(w http.ResponseWriter, r *http.Request) string {
	if controller.opts.AuthUserID != nil && controller.opts.AuthUserID(r) == "" {
		return shared.FlashOrRedirect(controller.opts, w, r, "You are not logged in. Please login to continue.", controller.opts.HomeURL, 10)
	}

	if controller.opts.Store == nil {
		return shared.FlashOrRedirect(controller.opts, w, r, "Stats store not available", controller.opts.HomeURL, 10)
	}

	ip := req.GetStringTrimmed(r, shared.ParamIP)
	if ip == "" {
		return shared.FlashOrRedirect(controller.opts, w, r, "No IP address specified", controller.opts.BaseURL, 10)
	}

	breadcrumbs := shared.Breadcrumbs([]shared.Breadcrumb{
		{Name: "Home", URL: controller.opts.HomeURL},
		{Name: "Stats", URL: controller.opts.BaseURL},
		{Name: "Visitors", URL: shared.NewLinks(controller.opts.BaseURL).Visitors(nil)},
		{Name: "IP: " + ip, URL: ""},
	})

	heading := hb.Heading1().HTML("IP Details: " + ip)

	linksHelper := shared.NewLinks(controller.opts.BaseURL)
	urlLoadPaths := linksHelper.IPDetails(map[string]string{
		"action": actionLoadPaths,
		"ip":     ip,
	})
	urlFlagBot := linksHelper.IPDetails(map[string]string{
		"action": actionFlagBot,
		"ip":     ip,
	})
	urlFlagThreat := linksHelper.IPDetails(map[string]string{
		"action": actionFlagThreat,
		"ip":     ip,
	})
	urlRemoveEntries := linksHelper.IPDetails(map[string]string{
		"action": actionRemoveEntries,
		"ip":     ip,
	})
	urlVisitors := linksHelper.Visitors(nil)
	urlSessions := linksHelper.Sessions(nil)
	urlDashboard := linksHelper.Dashboard(nil)
	urlSettings := linksHelper.Settings(nil)
	urlIPDetails := linksHelper.IPDetails(map[string]string{"ip": ip})

	html := strings.ReplaceAll(ipdetailsHTML, "urlLoadPaths", "'"+urlLoadPaths+"'")
	html = strings.ReplaceAll(html, "urlFlagBot", "'"+urlFlagBot+"'")
	html = strings.ReplaceAll(html, "urlFlagThreat", "'"+urlFlagThreat+"'")
	html = strings.ReplaceAll(html, "urlRemoveEntries", "'"+urlRemoveEntries+"'")
	html = strings.ReplaceAll(html, "__URL_DASHBOARD__", urlDashboard)
	html = strings.ReplaceAll(html, "__URL_VISITORS__", urlVisitors)
	html = strings.ReplaceAll(html, "__URL_SESSIONS__", urlSessions)
	html = strings.ReplaceAll(html, "__URL_SETTINGS__", urlSettings)
	html = strings.ReplaceAll(html, "__URL_IP_DETAILS__", urlIPDetails)
	html = strings.ReplaceAll(html, "__IP__", ip)
	js := strings.ReplaceAll(ipdetailsJS, "urlLoadPaths", "'"+urlLoadPaths+"'")
	js = strings.ReplaceAll(js, "urlFlagBot", "'"+urlFlagBot+"'")
	js = strings.ReplaceAll(js, "urlFlagThreat", "'"+urlFlagThreat+"'")
	js = strings.ReplaceAll(js, "urlRemoveEntries", "'"+urlRemoveEntries+"'")
	js = strings.ReplaceAll(js, "__IP__", ip)

	vueCDN := hb.Script("").Src(cdn.VueJs_3_5_32())

	content := hb.Div().
		Class("container").
		Child(heading).
		Child(breadcrumbs).
		Child(hb.HR()).
		Child(vueCDN).
		Child(hb.Raw(html)).
		Child(hb.Script(js))

	controller.opts.Layout.SetTitle("IP Details: " + ip + " | Stats")
	controller.opts.Layout.SetBody(content.ToHTML())
	controller.opts.Layout.SetScriptURLs([]string{cdn.Notiflix_3_2_8()})
	controller.opts.Layout.SetScripts([]string{})
	controller.opts.Layout.SetStyleURLs([]string{shared.FlagIconsCSS})
	controller.opts.Layout.SetStyles([]string{cdn.Notiflix_3_2_8_CSS()})

	return controller.opts.Layout.Render(w, r)
}
