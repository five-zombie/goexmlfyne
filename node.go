// Package goexml parses goexml UI documents into Fyne widgets.
//
// goexml is the declarative XML standard. This module, goexml-fyne,
// is its parser for the Fyne GUI toolkit.
package goexml

// Node is one element of a decoded goexml document.
type Node struct {
	// Type is the element name, e.g. "vbox" or "button".
	Type string
	// Attrs holds the element attributes by name.
	Attrs map[string]string
	// Children holds nested elements in document order.
	Children []*Node
	// Line is the 1-based source line of the start tag.
	Line int
	// Text is trimmed character content inside the element, or "".
	// Only cell and text elements may carry it; every other element
	// rejects non-empty Text during validation.
	Text string
}

// Attr returns the value of the named attribute, or "" when absent.
func (n *Node) Attr(key string) string {
	if n == nil {
		return ""
	}
	return n.Attrs[key]
}
