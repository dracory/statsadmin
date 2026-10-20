package shared

import (
	"github.com/dracory/hb"
)

// Breadcrumbs renders a breadcrumb navigation trail from the given entries.
// The last entry should have an empty URL to indicate the current page.
func Breadcrumbs(items []Breadcrumb) hb.TagInterface {
	ol := hb.Ol().Class("breadcrumb")

	for i, item := range items {
		li := hb.Li().Class("breadcrumb-item")
		if i == len(items)-1 {
			li = li.Class("active").Attr("aria-current", "page")
			li = li.Child(hb.Span().Text(item.Name))
		} else {
			li = li.Child(hb.A().Href(item.URL).Text(item.Name))
		}
		ol = ol.Child(li)
	}

	return hb.Nav().Attr("aria-label", "breadcrumb").Child(ol)
}
