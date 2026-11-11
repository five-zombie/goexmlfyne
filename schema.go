package goexml

// Version is the goexml standard version this parser implements.
const Version = "0.1.0"

// commonAttrs are allowed on every widget element.
var commonAttrs = []string{"id", "visible", "disabled", "style"}

// tagAttrs maps each 0.1.0 element to its own allowed attributes.
// Common attributes are merged in by allowedAttrs, except for the
// root, the window, and border region wrappers which take no
// common attributes.
var tagAttrs = map[string][]string{
	"goexml": {},
	"window": {"title", "size"},
	"vbox":   {"padding", "spacing", "align", "justify", "column", "min", "max"},
	"hbox":   {"padding", "spacing", "align", "justify"},
	"border": {},
	"top":    {},
	"bottom": {},
	"left":   {},
	"right":  {},
	"center": {},
	"label":  {"text", "expand"},
	"text":   {"v", "wrap"},
	"button": {"text", "onClick", "expand"},
	"entry":  {"text", "placeholder", "expand"},
	"rect":   {"color"},
	"select": {"onChange"},
	"option": {"value", "text", "selected"},
	"table":  {"onSelect"},
	"row":    {},
	"cell":   {"text"},
}

// withCommon lists the elements that accept common attributes.
var withCommon = map[string]bool{
	"vbox": true, "hbox": true, "border": true,
	"label": true, "button": true, "entry": true,
	"select": true, "table": true, "rect": true,
	"text": true,
}

// idRequired lists the elements that must carry a unique id.
var idRequired = map[string]bool{
	"vbox": true, "hbox": true, "border": true,
	"label": true, "button": true, "entry": true,
	"select": true, "table": true, "rect": true,
	"text": true,
}

// regions are the only valid children of a border element.
var regions = map[string]bool{
	"top": true, "bottom": true, "left": true, "right": true, "center": true,
}

// widgets are elements that render into Fyne objects.
var widgets = map[string]bool{
	"vbox": true, "hbox": true, "border": true,
	"label": true, "button": true, "entry": true,
	"select": true, "table": true, "rect": true,
	"text": true,
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
	if tag == "goexml" {
		set["version"] = true
	}
	return set
}
