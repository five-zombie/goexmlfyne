package goexml

import (
	"bytes"
	"fmt"
	"image"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
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
// against the same resource set the build used. An unknown key or a
// resource that is not a decodable image is an error, so a swapped
// image can never silently render empty; the min size follows the new
// resource's intrinsic pixels. It also fills a src-less slot image.
func (w *Window) SetImage(id, key string) error {
	n, ok := w.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "image" {
		return nodeErr(n, "cannot set an image on <%s>", n.Type)
	}
	res, ok := w.resources[key]
	if !ok {
		return fmt.Errorf("goexml: unknown image resource %q", key)
	}
	return w.setImage(id, res)
}

// SetImageRes swaps the resource shown by an <image> directly — no
// resource set lookup, so bytes fetched late (server, database,
// object store) can be attached without registering them first. The
// bytes must decode as an image, so a filled image never renders
// empty; the min size follows the resource's intrinsic pixels.
func (w *Window) SetImageRes(id string, res fyne.Resource) error {
	n, ok := w.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "image" {
		return nodeErr(n, "cannot set an image on <%s>", n.Type)
	}
	return w.setImage(id, res)
}

// setImage validates the resource and installs it on the image,
// updating the min size to the new resource's intrinsic pixels.
func (w *Window) setImage(id string, res fyne.Resource) error {
	if res == nil {
		return fmt.Errorf("goexml: nil resource for image %q", id)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(res.Content()))
	if err != nil {
		return fmt.Errorf("goexml: bad image resource %q: %v", res.Name(), err)
	}
	img := w.byID[id].(*canvas.Image)
	img.Resource = res
	img.SetMinSize(fyne.NewSize(float32(cfg.Width), float32(cfg.Height)))
	img.Refresh()
	return nil
}

// UnfilledImages returns the ids of slot images (no src) that no
// SetImage or SetImageRes has filled yet, sorted for determinism. An
// unfilled slot renders empty and invisible like an unfilled container
// slot; call this before the window shows to catch orphans.
func (w *Window) UnfilledImages() []string {
	var ids []string
	for id, n := range w.nodes {
		if n.Type != "image" || n.Attr("src") != "" {
			continue
		}
		if img, ok := w.byID[id].(*canvas.Image); ok && img.Resource == nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// SetText sets the text of a label, text, button or entry. Setting a
// plain text removes the {{placeholder}} template registered for the
// id, so later SetTextFormat calls are rejected — the last call wins.
func (w *Window) SetText(id, v string) error {
	n, ok := w.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	switch n.Type {
	case "label", "text":
		w.byID[id].(*widget.Label).SetText(v)
	case "button":
		w.byID[id].(*widget.Button).SetText(v)
	case "entry":
		w.byID[id].(*widget.Entry).SetText(v)
	default:
		return nodeErr(n, "cannot set text on <%s>", n.Type)
	}
	delete(w.templates, id)
	return nil
}

// Text returns the current text of a label, text, button or entry.
// Entry is two-way: its value reflects what the user typed.
func (w *Window) Text(id string) (string, error) {
	n, ok := w.nodes[id]
	if !ok {
		return "", fmt.Errorf("goexml: no widget with id %q", id)
	}
	switch n.Type {
	case "label", "text":
		return w.byID[id].(*widget.Label).Text, nil
	case "button":
		return w.byID[id].(*widget.Button).Text, nil
	case "entry":
		return w.byID[id].(*widget.Entry).Text, nil
	default:
		return "", nodeErr(n, "has no text to read on <%s>", n.Type)
	}
}

// SetValue selects the option carrying value v on a select.
func (w *Window) SetValue(id, v string) error {
	n, ok := w.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "select" {
		return nodeErr(n, "cannot set value on <%s>", n.Type)
	}
	s := w.byID[id].(*widget.Select)
	for label, value := range w.selectValues[id] {
		if value == v {
			// Direct assignment plus refresh, mirroring Build: firing
			// OnChanged here would call the handler outside an event.
			s.Selected = label
			s.Refresh()
			return nil
		}
	}
	return nodeErr(n, "no option with value %q", v)
}

// SetTextFormat renders the {{placeholder}} template the XML declared
// for this id with vals and sets it as the widget text. It can be
// called repeatedly; each call re-renders from the raw template.
func (w *Window) SetTextFormat(id string, vals map[string]string) error {
	raw, ok := w.templates[id]
	if !ok {
		return fmt.Errorf("goexml: no {{placeholder}} template for id %q", id)
	}
	n := w.nodes[id]
	out, err := renderTemplate(raw, vals)
	if err != nil {
		return nodeErr(n, "render: %v", err)
	}
	switch n.Type {
	case "label", "text":
		w.byID[id].(*widget.Label).SetText(out)
	case "button":
		w.byID[id].(*widget.Button).SetText(out)
	case "entry":
		w.byID[id].(*widget.Entry).SetText(out)
	default:
		return nodeErr(n, "cannot set text on <%s>", n.Type)
	}
	return nil
}

// SetValueFormat renders the {{placeholder}} templates in the option
// labels of a select and updates its option list. The current
// selection is kept by value.
func (w *Window) SetValueFormat(id string, vals map[string]string) error {
	n, ok := w.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "select" {
		return nodeErr(n, "cannot set value on <%s>", n.Type)
	}
	raws, ok := w.optionRaw[id]
	if !ok {
		return fmt.Errorf("goexml: no {{placeholder}} template for id %q", id)
	}
	s := w.byID[id].(*widget.Select)
	prev := w.selectValues[id][s.Selected]
	if prev == "" {
		prev = s.Selected
	}
	labels := make([]string, len(raws))
	m := map[string]string{}
	for i, p := range raws {
		label, err := renderTemplate(p.raw, vals)
		if err != nil {
			return nodeErr(n, "option %d: %v", i, err)
		}
		labels[i] = label
		m[label] = p.value
	}
	w.selectValues[id] = m
	s.Options = labels
	for _, label := range labels {
		if m[label] == prev {
			s.Selected = label
			break
		}
	}
	s.Refresh()
	return nil
}

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
