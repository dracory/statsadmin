package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dracory/cdn"
	"github.com/dracory/hb"
	"github.com/dracory/statsadmin"
	"github.com/dracory/statsstore"
	"github.com/dromara/carbon/v2"
	_ "modernc.org/sqlite"
)

// funcLayout renders a full HTML page with Bootstrap, Bootstrap Icons, and
// Chart.js, wrapped around the statsadmin content.
func funcLayout(w http.ResponseWriter, r *http.Request, title string, body string, options statsadmin.LayoutOptions) string {
	if title == "" {
		title = "Visitor Analytics"
	}

	page := hb.NewWebpage().
		SetLanguage("en").
		SetTitle(title).
		StyleURL(cdn.BootstrapCss_5_3_3()).
		StyleURL(cdn.BootstrapIconsCss_1_11_3()).
		StyleURL("https://cdn.jsdelivr.net/gh/lipis/flag-icons@7.2.3/css/flag-icons.min.css")

	for _, url := range options.StyleURLs {
		page = page.StyleURL(url)
	}
	for _, css := range options.Styles {
		page = page.Style(css)
	}

	page = page.ScriptURL(cdn.BootstrapJs_5_3_3())
	page = page.ScriptURL("https://cdn.jsdelivr.net/npm/chart.js")
	for _, url := range options.ScriptURLs {
		page = page.ScriptURL(url)
	}
	for _, js := range options.Scripts {
		page = page.Script(js)
	}

	page.Body().
		Class("bg-light").
		Child(hb.Div().
			Class("container-fluid py-4").
			Child(hb.Raw(body)))

	return page.ToHTML()
}

// countryNameByIso2 provides a simple ISO 3166-1 alpha-2 to country name
// mapping for the demo.  In production this would use a proper locale
// library.
func countryNameByIso2(code string) (string, error) {
	names := map[string]string{
		"US": "United States",
		"DE": "Germany",
		"IN": "India",
		"CA": "Canada",
		"GB": "United Kingdom",
		"IT": "Italy",
		"FR": "France",
		"AU": "Australia",
		"NL": "Netherlands",
		"BR": "Brazil",
		"JP": "Japan",
		"ES": "Spain",
		"MX": "Mexico",
		"RU": "Russia",
		"CN": "China",
		"SE": "Sweden",
		"CH": "Switzerland",
		"PL": "Poland",
		"SG": "Singapore",
		"IE": "Ireland",
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	if name, ok := names[code]; ok {
		return name, nil
	}
	return code, nil
}

// seedData populates the store with realistic demo visitor records spread
// across the last 7 days.
func seedData(store statsstore.StoreInterface) error {
	ctx := context.Background()
	now := carbon.Now(carbon.UTC)

	type seedVisitor struct {
		offsetHours int
		path        string
		country     string
		browser     string
		browserVer  string
		os          string
		osVer       string
		device      string
		deviceType  string
		referrer    string
		ip          string
		fingerprint string
	}

	visitors := []seedVisitor{
		// Today
		{0, "/", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.1", "fp001"},
		{-1, "/docs", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.1", "fp001"},
		{-2, "/pricing", "DE", "Firefox", "121.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://github.com/dracory/statsstore", "198.51.100.2", "fp002"},
		{-3, "/", "IN", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=web+analytics", "192.0.2.3", "fp003"},
		{-4, "/event/signup", "IN", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=web+analytics", "192.0.2.3", "fp003"},
		{-5, "/", "GB", "Safari", "17.2", "iOS", "17.2", "iPhone 15", "mobile", "", "198.18.0.4", "fp004"},
		{-6, "/docs/self-hosting", "GB", "Safari", "17.2", "iOS", "17.2", "iPhone 15", "mobile", "", "198.18.0.4", "fp004"},
		{-7, "/", "CA", "Edge", "120.0", "Windows", "11", "Desktop", "desktop", "https://bing.com/search?q=stats", "203.0.113.5", "fp005"},
		{-8, "/features", "CA", "Edge", "120.0", "Windows", "11", "Desktop", "desktop", "https://bing.com/search?q=stats", "203.0.113.5", "fp005"},
		{-10, "/", "FR", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=self+hosted+analytics", "198.51.100.6", "fp006"},
		{-11, "/docs/script", "FR", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=self+hosted+analytics", "198.51.100.6", "fp006"},
		{-12, "/event/demo", "IT", "Mobile Safari", "17.2", "iOS", "17.1", "iPhone 14", "mobile", "https://producthunt.com/posts/statsstore", "192.0.2.7", "fp007"},
		{-14, "/", "AU", "Chrome", "119.0", "Windows", "10", "Desktop", "desktop", "https://google.com/search?q=visitor+analytics", "203.0.113.8", "fp008"},
		{-15, "/docs/roadmap", "AU", "Chrome", "119.0", "Windows", "10", "Desktop", "desktop", "https://google.com/search?q=visitor+analytics", "203.0.113.8", "fp008"},
		{-16, "/", "NL", "Firefox", "121.0", "Linux", "Fedora 39", "Desktop", "desktop", "https://github.com/dracory/statsstore", "198.51.100.9", "fp009"},
		{-20, "/", "BR", "Mobile Chrome", "120.0", "Android", "13", "Galaxy S23", "mobile", "https://duckduckgo.com/?q=open+source+analytics", "192.0.2.10", "fp010"},
		{-21, "/pricing", "BR", "Mobile Chrome", "120.0", "Android", "13", "Galaxy S23", "mobile", "https://duckduckgo.com/?q=open+source+analytics", "192.0.2.10", "fp010"},
		{-22, "/event/signup", "BR", "Mobile Chrome", "120.0", "Android", "13", "Galaxy S23", "mobile", "https://duckduckgo.com/?q=open+source+analytics", "192.0.2.10", "fp010"},
		{-24, "/", "JP", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://google.com/search?q=analytics+tool", "203.0.113.11", "fp011"},
		{-25, "/docs", "JP", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://google.com/search?q=analytics+tool", "203.0.113.11", "fp011"},
		{-26, "/docs/self-hosting", "JP", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://google.com/search?q=analytics+tool", "203.0.113.11", "fp011"},
		{-30, "/", "ES", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=web+stats", "198.51.100.12", "fp012"},
		{-31, "/event/demo", "ES", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=web+stats", "198.51.100.12", "fp012"},
		{-32, "/event/signup", "ES", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=web+stats", "198.51.100.12", "fp012"},
		{-34, "/", "MX", "Mobile Safari", "16.6", "iOS", "16.6", "iPhone 13", "mobile", "", "192.0.2.13", "fp013"},
		{-35, "/features", "MX", "Mobile Safari", "16.6", "iOS", "16.6", "iPhone 13", "mobile", "", "192.0.2.13", "fp013"},
		{-36, "/", "SE", "Chrome", "120.0", "Linux", "Arch", "Desktop", "desktop", "https://github.com/dracory/statsstore", "203.0.113.14", "fp014"},
		{-37, "/docs/track-events", "SE", "Chrome", "120.0", "Linux", "Arch", "Desktop", "desktop", "https://github.com/dracory/statsstore", "203.0.113.14", "fp014"},
		{-40, "/", "CH", "Firefox", "121.0", "Windows", "11", "Desktop", "desktop", "https://alternativeto.net/software/statsstore", "198.51.100.15", "fp015"},
		{-41, "/docs/self-host-vs-cloud", "CH", "Firefox", "121.0", "Windows", "11", "Desktop", "desktop", "https://alternativeto.net/software/statsstore", "198.51.100.15", "fp015"},
		{-42, "/", "PL", "Chrome", "120.0", "Windows", "10", "Desktop", "desktop", "https://google.com/search?q=privacy+analytics", "203.0.113.16", "fp016"},
		{-44, "/", "SG", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=analytics", "192.0.2.17", "fp017"},
		{-45, "/pricing", "SG", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=analytics", "192.0.2.17", "fp017"},
		{-46, "/event/signup", "SG", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=analytics", "192.0.2.17", "fp017"},
		{-48, "/", "IE", "Safari", "17.2", "macOS", "14.1", "MacBook Pro", "desktop", "https://producthunt.com/posts/statsstore", "198.51.100.18", "fp018"},
		{-49, "/docs", "IE", "Safari", "17.2", "macOS", "14.1", "MacBook Pro", "desktop", "https://producthunt.com/posts/statsstore", "198.51.100.18", "fp018"},
		{-50, "/docs/self-hosting-guides/self-hosting-manual", "IE", "Safari", "17.2", "macOS", "14.1", "MacBook Pro", "desktop", "https://producthunt.com/posts/statsstore", "198.51.100.18", "fp018"},
		{-52, "/", "DE", "Opera", "107.0", "Windows", "11", "Desktop", "desktop", "https://trustmrr.com/review/statsstore", "203.0.113.19", "fp019"},
		{-53, "/features", "DE", "Opera", "107.0", "Windows", "11", "Desktop", "desktop", "https://trustmrr.com/review/statsstore", "203.0.113.19", "fp019"},
		{-54, "/", "RU", "Chrome", "120.0", "Windows", "10", "Desktop", "desktop", "https://openalternative.co/products/statsstore", "198.51.100.20", "fp020"},
		{-55, "/docs/script", "RU", "Chrome", "120.0", "Windows", "10", "Desktop", "desktop", "https://openalternative.co/products/statsstore", "198.51.100.20", "fp020"},
		{-56, "/event/hello-world", "RU", "Chrome", "120.0", "Windows", "10", "Desktop", "desktop", "https://openalternative.co/products/statsstore", "198.51.100.20", "fp020"},
		{-58, "/", "CN", "Mobile Firefox", "121.0", "Android", "12", "Redmi Note 12", "mobile", "https://selfh.st/statsstore", "192.0.2.21", "fp021"},
		{-60, "/", "US", "Googlebot", "2.1", "Linux", "", "Bot", "bot", "", "66.249.66.1", "fpbot1"},
		{-61, "/docs", "US", "Googlebot", "2.1", "Linux", "", "Bot", "bot", "", "66.249.66.1", "fpbot1"},
		{-62, "/robots.txt", "US", "Bingbot", "2.0", "Windows", "", "Bot", "bot", "", "157.55.39.1", "fpbot2"},
		{-63, "/sitemap.xml", "DE", "AhrefsBot", "7.0", "Linux", "", "Bot", "bot", "", "54.36.148.1", "fpbot3"},
		{-64, "/", "CA", "Android Browser", "4.4", "Android", "12", "Galaxy A14", "mobile", "", "192.0.2.23", "fp023"},
		{-64, "/", "GB", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+tracking", "203.0.113.24", "fp024"},
		{-65, "/docs", "GB", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+tracking", "203.0.113.24", "fp024"},
		{-66, "/docs/roadmap", "GB", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+tracking", "203.0.113.24", "fp024"},
		{-68, "/", "FR", "Mobile Safari", "17.2", "iOS", "17.2", "iPhone 15", "mobile", "https://bing.com/search?q=analytics", "192.0.2.25", "fp025"},
		{-70, "/", "IT", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=stats+dashboard", "198.51.100.26", "fp026"},
		{-71, "/pricing", "IT", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=stats+dashboard", "198.51.100.26", "fp026"},
		{-72, "/event/demo", "IT", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=stats+dashboard", "198.51.100.26", "fp026"},
		{-74, "/", "AU", "Edge", "120.0", "Windows", "11", "Desktop", "desktop", "https://duckduckgo.com/?q=web+analytics+tool", "203.0.113.27", "fp027"},
		{-76, "/", "IN", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=open+source+analytics", "198.51.100.28", "fp028"},
		{-77, "/docs/self-hosting", "IN", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=open+source+analytics", "198.51.100.28", "fp028"},
		{-78, "/docs/self-host-vs-cloud", "IN", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=open+source+analytics", "198.51.100.28", "fp028"},
		{-80, "/", "US", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://google.com/search?q=analytics+dashboard", "192.0.2.29", "fp029"},
		{-81, "/event/signup", "US", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://google.com/search?q=analytics+dashboard", "192.0.2.29", "fp029"},
		{-82, "/", "DE", "Firefox", "121.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://github.com/dracory/statsstore", "203.0.113.30", "fp030"},
		{-83, "/docs", "DE", "Firefox", "121.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://github.com/dracory/statsstore", "203.0.113.30", "fp030"},
		{-84, "/docs/track-events", "DE", "Firefox", "121.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://github.com/dracory/statsstore", "203.0.113.30", "fp030"},
		{-86, "/", "NL", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=privacy+first+analytics", "198.51.100.31", "fp031"},
		{-88, "/", "BR", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://producthunt.com/posts/statsstore", "203.0.113.32", "fp032"},
		{-89, "/features", "BR", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://producthunt.com/posts/statsstore", "203.0.113.32", "fp032"},
		{-90, "/event/custom-event", "BR", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://producthunt.com/posts/statsstore", "203.0.113.32", "fp032"},
		{-92, "/", "JP", "Mobile Safari", "17.2", "iOS", "17.2", "iPhone 15", "mobile", "", "192.0.2.33", "fp033"},
		{-94, "/", "ES", "Chrome", "120.0", "Windows", "10", "Desktop", "desktop", "https://google.com/search?q=stats", "198.51.100.34", "fp034"},
		{-95, "/docs/self-hosting", "ES", "Chrome", "120.0", "Windows", "10", "Desktop", "desktop", "https://google.com/search?q=stats", "198.51.100.34", "fp034"},
		{-96, "/", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.35", "fp036"},
		{-97, "/docs", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.35", "fp036"},
		{-98, "/pricing", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.35", "fp036"},
		{-99, "/event/signup", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.35", "fp036"},
		{-100, "/event/demo", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.35", "fp036"},
		{-102, "/", "CA", "Firefox", "121.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://duckduckgo.com/?q=self+hosted", "198.51.100.37", "fp038"},
		{-104, "/", "GB", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+stats", "203.0.113.38", "fp039"},
		{-105, "/docs/roadmap", "GB", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+stats", "203.0.113.38", "fp039"},
		{-106, "/features", "GB", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+stats", "203.0.113.38", "fp039"},
		{-108, "/", "FR", "Edge", "120.0", "Windows", "11", "Desktop", "desktop", "https://bing.com/search?q=analytics+tool", "198.51.100.39", "fp040"},
		{-110, "/", "IT", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://google.com/search?q=analytics", "192.0.2.40", "fp041"},
		{-111, "/pricing", "IT", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://google.com/search?q=analytics", "192.0.2.40", "fp041"},
		{-112, "/", "AU", "Safari", "17.2", "macOS", "14.2", "MacBook Pro", "desktop", "https://github.com/dracory/statsstore", "203.0.113.41", "fp042"},
		{-113, "/docs", "AU", "Safari", "17.2", "macOS", "14.2", "MacBook Pro", "desktop", "https://github.com/dracory/statsstore", "203.0.113.41", "fp042"},
		{-114, "/docs/self-hosting", "AU", "Safari", "17.2", "macOS", "14.2", "MacBook Pro", "desktop", "https://github.com/dracory/statsstore", "203.0.113.41", "fp042"},
		{-116, "/", "IN", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=web+analytics", "198.51.100.43", "fp044"},
		{-117, "/event/signup", "IN", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=web+analytics", "198.51.100.43", "fp044"},
		{-118, "/event/demo", "IN", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=web+analytics", "198.51.100.43", "fp044"},
		{-120, "/", "DE", "Mobile Firefox", "121.0", "Android", "13", "Galaxy S23", "mobile", "https://duckduckgo.com/?q=analytics", "192.0.2.45", "fp046"},
		{-122, "/", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.47", "fp048"},
		{-123, "/docs", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.47", "fp048"},
		{-124, "/docs/self-hosting", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.47", "fp048"},
		{-125, "/docs/self-host-vs-cloud", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.47", "fp048"},
		{-126, "/pricing", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.47", "fp048"},
		{-128, "/", "SE", "Chrome", "120.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://github.com/dracory/statsstore", "198.51.100.49", "fp050"},
		{-130, "/", "PL", "Firefox", "121.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.51", "fp052"},
		{-131, "/features", "PL", "Firefox", "121.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.51", "fp052"},
		{-132, "/event/signup", "PL", "Firefox", "121.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics", "203.0.113.51", "fp052"},
		{-134, "/", "CH", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://alternativeto.net/software/statsstore", "198.51.100.53", "fp054"},
		{-135, "/docs/track-events", "CH", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://alternativeto.net/software/statsstore", "198.51.100.53", "fp054"},
		{-136, "/", "SG", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=visitor+analytics", "203.0.113.55", "fp056"},
		{-138, "/", "IE", "Mobile Safari", "17.2", "iOS", "17.2", "iPhone 15", "mobile", "", "192.0.2.57", "fp058"},
		{-139, "/pricing", "IE", "Mobile Safari", "17.2", "iOS", "17.2", "iPhone 15", "mobile", "", "192.0.2.57", "fp058"},
		{-140, "/", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics+dashboard", "203.0.113.59", "fp060"},
		{-141, "/docs", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics+dashboard", "203.0.113.59", "fp060"},
		{-142, "/docs/self-hosting", "US", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=analytics+dashboard", "203.0.113.59", "fp060"},
		{-144, "/", "CA", "Edge", "120.0", "Windows", "11", "Desktop", "desktop", "https://bing.com/search?q=stats", "198.51.100.61", "fp062"},
		{-146, "/", "GB", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=analytics", "192.0.2.63", "fp064"},
		{-147, "/event/demo", "GB", "Mobile Chrome", "120.0", "Android", "14", "Pixel 8", "mobile", "https://duckduckgo.com/?q=analytics", "192.0.2.63", "fp064"},
		{-148, "/", "FR", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.65", "fp066"},
		{-149, "/features", "FR", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.65", "fp066"},
		{-150, "/event/signup", "FR", "Chrome", "120.0", "Windows", "11", "Desktop", "desktop", "https://google.com/search?q=statsstore", "203.0.113.65", "fp066"},
		{-152, "/", "IT", "Firefox", "121.0", "Linux", "Ubuntu 22.04", "Desktop", "desktop", "https://github.com/dracory/statsstore", "198.51.100.67", "fp068"},
		{-154, "/", "AU", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=analytics", "203.0.113.69", "fp070"},
		{-155, "/docs", "AU", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=analytics", "203.0.113.69", "fp070"},
		{-156, "/docs/roadmap", "AU", "Chrome", "120.0", "macOS", "14.2", "MacBook Pro", "desktop", "https://google.com/search?q=analytics", "203.0.113.69", "fp070"},
		{-158, "/", "NL", "Safari", "17.2", "macOS", "14.2", "MacBook Air", "desktop", "https://producthunt.com/posts/statsstore", "198.51.100.71", "fp072"},
	}

	for _, sv := range visitors {
		createdAt := now.AddHours(sv.offsetHours * -1).ToDateTimeString(carbon.UTC)
		botVal := statsstore.VALUE_NO
		if strings.Contains(strings.ToLower(sv.browser), "bot") || strings.Contains(strings.ToLower(sv.path), "robot") || strings.Contains(strings.ToLower(sv.path), "sitemap") {
			botVal = statsstore.VALUE_YES
		}
		v := statsstore.NewVisitor().
			SetCountry(sv.country).
			SetPath(sv.path).
			SetIpAddress(sv.ip).
			SetFingerprint(sv.fingerprint).
			SetUserBrowser(sv.browser).
			SetUserBrowserVersion(sv.browserVer).
			SetUserOs(sv.os).
			SetUserOsVersion(sv.osVer).
			SetUserDevice(sv.device).
			SetUserDeviceType(sv.deviceType).
			SetUserReferrer(sv.referrer).
			SetUserAgent(sv.browser + "/" + sv.browserVer + " (" + sv.os + " " + sv.osVer + ")").
			SetBot(botVal).
			SetCreatedAt(createdAt)

		if err := store.VisitorCreate(ctx, v); err != nil {
			return fmt.Errorf("failed to seed visitor: %w", err)
		}
	}

	return nil
}

func main() {
	// Open in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:?parseTime=true")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Create the store with auto-migration
	store, err := statsstore.NewStore(statsstore.NewStoreOptions{
		DB:                   db,
		VisitorTableName:     "visitors",
		AutomigrateEnabled:   true,
		ExcludedPathPrefixes: []string{"/admin/"},
	})
	if err != nil {
		log.Fatalf("failed to create store: %v", err)
	}

	// Seed the store with demo data
	if err := seedData(store); err != nil {
		log.Fatalf("failed to seed data: %v", err)
	}

	log.Printf("Seeded visitor demo data")

	// Create the admin handler (stateless — safe to build once and reuse)
	handler, err := statsadmin.New(statsadmin.AdminOptions{
		Store:             store,
		FuncLayout:        funcLayout,
		HomeURL:           "/",
		WebsiteUrl:        "https://example.com",
		BaseURL:           "/",
		CountryNameByIso2: countryNameByIso2,
	})
	if err != nil {
		log.Fatalf("failed to create stats admin: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	})

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	fmt.Printf("Admin demo server starting on http://localhost%s\n", addr)
	fmt.Printf("Open http://localhost%s/admin/home in your browser\n", addr)
	fmt.Println("Press Ctrl+C to stop")

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
