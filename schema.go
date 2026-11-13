package goexml

// Version is the goexml standard version this parser implements.
const Version = "0.1.0"

// commonAttrs are allowed on every widget element. disabled is not
// common: it only reaches widgets that implement fyne.Disableable,
// so the schema lists it per tag instead of promising it everywhere.
var commonAttrs = []string{"id", "visible", "style"}

// tagAttrs maps each 0.1.0 element to its own allowed attributes.
// Common attributes are merged in by allowedAttrs, except for the
// root and the window, which take no common attributes.
var tagAttrs = map[string][]string{
	"goexml":      {},
	"goexml-item": {},
	"window":      {"title", "size"},
	"container":   {},
	"slot":        {"style"},
	"vbox":        {"spacing", "align", "justify", "column", "min", "max"},
	"hbox":        {"spacing", "align", "justify"},
	"label":       {"text"},
	"text":        {"v", "wrap"},
	"button":      {"text", "onClick", "disabled"},
	"entry":       {"text", "placeholder", "disabled"},
	"rect":        {"color"},
	"image":       {"src", "fit"},
	"vbox-expand": {"spacing", "align", "justify", "column", "min", "max", "scrollable"},
	"select":      {"onChange", "disabled"},
	"option":      {"value", "text", "selected"},
	"table":       {"onSelect"},
	"row":         {},
	"cell":        {"text"},
}

// withCommon lists the elements that accept common attributes.
var withCommon = map[string]bool{
	"container": true,
	"vbox":      true, "hbox": true,
	"label": true, "button": true, "entry": true,
	"select": true, "table": true, "rect": true,
	"text": true, "image": true, "vbox-expand": true,
}

// idRequired lists the elements that must carry a unique id: the ones
// Go must reference by name because they cannot be static — a
// container's slot is filled through the contents map, and a
// vbox-expand's handle comes from Find. Every other element is
// addressable when it needs to be: id is optional for possibly-static
// widgets, and conditionally required by decode (a src-less slot image
// and any element carrying a {{placeholder}} template).
var idRequired = map[string]bool{
	"container":   true,
	"vbox-expand": true,
}

// widgets are elements that render into Fyne objects.
var widgets = map[string]bool{
	"container": true, "slot": true,
	"vbox": true, "hbox": true,
	"label": true, "button": true, "entry": true,
	"select": true, "table": true, "rect": true,
	"text": true, "image": true, "vbox-expand": true,
}

// allowedAttrs returns the full attribute set for a tag.
func allowedAttrs(tag string) map[string]bool {
	set := map[string]bool{}
	for _, a := range tagAttrs[tag] {
		set[a] = true
	}
	if withCommon[tag] {
		for _, a := range commonAttrs {
			set[a] = true
		}
	}
	// Both envelopes are roots: goexml is the document root, goexml-item
	// the fragment root. They are never legal as children, so they stay
	// out of widgets and every child list.
	if tag == "goexml" || tag == "goexml-item" {
		set["version"] = true
	}
	return set
}
