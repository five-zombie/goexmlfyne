package goexml

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
)

// Container is a built <container>: the XML design plus the slot that
// Go fills with content. Find returns it for the container's id, so the
// application can inject objects the XML schema does not know about.
type Container struct {
	*fyne.Container // holds the design root object
	node            *Node
	slot            *fyne.Container
}

// SetContent places obj in the slot, replacing any previous content.
// The design around the slot (header, footer, whatever the XML holds)
// is never touched.
func (c *Container) SetContent(obj fyne.CanvasObject) {
	c.slot.Objects = []fyne.CanvasObject{obj}
	c.slot.Refresh()
}

// SetImage swaps the resource shown by an <image>, resolving key
// renderTemplate substitutes every {{token}} with vals[token]. Decode
// already validated the token shapes, so rendering only fails when a
// value is missing. There is no logic and no escaping: text before the
// first {{, and literal {{ without }} (hand-built trees only), are
// rejected or kept as-is respectively.
func renderTemplate(s string, vals map[string]string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); {
		open := strings.Index(s[i:], "{{")
		if open < 0 {
			b.WriteString(s[i:])
			break
		}
		b.WriteString(s[i : i+open])
		rest := s[i+open+2:]
		close := strings.Index(rest, "}}")
		if close < 0 {
			return "", fmt.Errorf("unclosed placeholder in %q", s)
		}
		tok := rest[:close]
		v, ok := vals[tok]
		if !ok {
			return "", fmt.Errorf("missing placeholder %q", tok)
		}
		b.WriteString(v)
		i += open + 2 + close + 2
	}
	return b.String(), nil
}
