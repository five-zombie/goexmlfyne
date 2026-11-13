package goexml

import (
	"bytes"
	"fmt"
	"image"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// widgetScope is the widget-facing registry a Window and an appended
// list item both front: the id maps the setters dispatch on. A Window
// owns one scope for the whole document; each appended list item owns
// a private scope, so fragment ids never collide across items.
type widgetScope struct {
	// resources is the resource set image src keys resolve against.
	resources map[string]fyne.Resource
	// byID maps widget ids to their built objects.
	byID map[string]fyne.CanvasObject
	// nodes maps ids to their XML nodes, so setters can dispatch by
	// widget kind.
	nodes map[string]*Node
	// containers maps container ids to their built handle.
	containers map[string]*Container
	// templates maps ids to their raw {{placeholder}} template text.
	templates map[string]string
	// selectValues maps a select id to its option text-to-value table,
	// so SelectValue can return the option value instead of the label.
	selectValues map[string]map[string]string
	// optionRaw stores a select's raw option labels when any of them
	// carries a {{placeholder}} template.
	optionRaw map[string][]optPair
}

// SetImage swaps the resource shown by an <image>, resolving key
// against the same resource set the build used. An unknown key or a
// resource that is not a decodable image is an error, so a swapped
// image can never silently render empty; the min size follows the new
// resource's intrinsic pixels. It also fills a src-less slot image.
func (s *widgetScope) SetImage(id, key string) error {
	n, ok := s.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "image" {
		return nodeErr(n, "cannot set an image on <%s>", n.Type)
	}
	res, ok := s.resources[key]
	if !ok {
		return fmt.Errorf("goexml: unknown image resource %q", key)
	}
	return s.setImage(id, res)
}

// SetImageRes swaps the resource shown by an <image> directly — no
// resource set lookup, so bytes fetched late (server, database,
// object store) can be attached without registering them first. The
// bytes must decode as an image, so a filled image never renders
// empty; the min size follows the resource's intrinsic pixels.
func (s *widgetScope) SetImageRes(id string, res fyne.Resource) error {
	n, ok := s.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "image" {
		return nodeErr(n, "cannot set an image on <%s>", n.Type)
	}
	return s.setImage(id, res)
}

// setImage validates the resource and installs it on the image,
// updating the min size to the new resource's intrinsic pixels.
func (s *widgetScope) setImage(id string, res fyne.Resource) error {
	if res == nil {
		return fmt.Errorf("goexml: nil resource for image %q", id)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(res.Content()))
	if err != nil {
		return fmt.Errorf("goexml: bad image resource %q: %v", res.Name(), err)
	}
	img := s.byID[id].(*canvas.Image)
	img.Resource = res
	img.SetMinSize(fyne.NewSize(float32(cfg.Width), float32(cfg.Height)))
	img.Refresh()
	return nil
}

// UnfilledImages returns the ids of slot images (no src) that no
// SetImage or SetImageRes has filled yet, sorted for determinism. An
// unfilled slot renders empty and invisible like an unfilled container
// slot; call this before the window shows to catch orphans.
func (s *widgetScope) UnfilledImages() []string {
	var ids []string
	for id, n := range s.nodes {
		if n.Type != "image" || n.Attr("src") != "" {
			continue
		}
		if img, ok := s.byID[id].(*canvas.Image); ok && img.Resource == nil {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// SetText sets the text of a label, text, button or entry. Setting a
// plain text removes the {{placeholder}} template registered for the
// id, so later SetTextFormat calls are rejected — the last call wins.
func (s *widgetScope) SetText(id, v string) error {
	n, ok := s.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	switch n.Type {
	case "label", "text":
		s.byID[id].(*widget.Label).SetText(v)
	case "button":
		s.byID[id].(*widget.Button).SetText(v)
	case "entry":
		s.byID[id].(*widget.Entry).SetText(v)
	default:
		return nodeErr(n, "cannot set text on <%s>", n.Type)
	}
	delete(s.templates, id)
	return nil
}

// Text returns the current text of a label, text, button or entry.
// Entry is two-way: its value reflects what the user typed.
func (s *widgetScope) Text(id string) (string, error) {
	n, ok := s.nodes[id]
	if !ok {
		return "", fmt.Errorf("goexml: no widget with id %q", id)
	}
	switch n.Type {
	case "label", "text":
		return s.byID[id].(*widget.Label).Text, nil
	case "button":
		return s.byID[id].(*widget.Button).Text, nil
	case "entry":
		return s.byID[id].(*widget.Entry).Text, nil
	default:
		return "", nodeErr(n, "has no text to read on <%s>", n.Type)
	}
}

// SetValue selects the option carrying value v on a select.
func (s *widgetScope) SetValue(id, v string) error {
	n, ok := s.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "select" {
		return nodeErr(n, "cannot set value on <%s>", n.Type)
	}
	sel := s.byID[id].(*widget.Select)
	for label, value := range s.selectValues[id] {
		if value == v {
			// Direct assignment plus refresh, mirroring Build: firing
			// OnChanged here would call the handler outside an event.
			sel.Selected = label
			sel.Refresh()
			return nil
		}
	}
	return nodeErr(n, "no option with value %q", v)
}

// SetTextFormat renders the {{placeholder}} template the XML declared
// for this id with vals and sets it as the widget text. It can be
// called repeatedly; each call re-renders from the raw template.
func (s *widgetScope) SetTextFormat(id string, vals map[string]string) error {
	raw, ok := s.templates[id]
	if !ok {
		return fmt.Errorf("goexml: no {{placeholder}} template for id %q", id)
	}
	n := s.nodes[id]
	out, err := renderTemplate(raw, vals)
	if err != nil {
		return nodeErr(n, "render: %v", err)
	}
	switch n.Type {
	case "label", "text":
		s.byID[id].(*widget.Label).SetText(out)
	case "button":
		s.byID[id].(*widget.Button).SetText(out)
	case "entry":
		s.byID[id].(*widget.Entry).SetText(out)
	default:
		return nodeErr(n, "cannot set text on <%s>", n.Type)
	}
	return nil
}

// SetValueFormat renders the {{placeholder}} templates in the option
// labels of a select and updates its option list. The current
// selection is kept by value.
func (s *widgetScope) SetValueFormat(id string, vals map[string]string) error {
	n, ok := s.nodes[id]
	if !ok {
		return fmt.Errorf("goexml: no widget with id %q", id)
	}
	if n.Type != "select" {
		return nodeErr(n, "cannot set value on <%s>", n.Type)
	}
	raws, ok := s.optionRaw[id]
	if !ok {
		return fmt.Errorf("goexml: no {{placeholder}} template for id %q", id)
	}
	sel := s.byID[id].(*widget.Select)
	prev := s.selectValues[id][sel.Selected]
	if prev == "" {
		prev = sel.Selected
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
	s.selectValues[id] = m
	sel.Options = labels
	for _, label := range labels {
		if m[label] == prev {
			sel.Selected = label
			break
		}
	}
	sel.Refresh()
	return nil
}

// SelectValue returns the option value of the current choice of a
// select widget. It reports false when id is unknown or not a select.
// A choice set outside the declared options falls back to its label.
func (s *widgetScope) SelectValue(id string) (string, bool) {
	o, ok := s.byID[id]
	if !ok {
		return "", false
	}
	sel, ok := o.(*widget.Select)
	if !ok {
		return "", false
	}
	if m, ok := s.selectValues[id]; ok {
		if v, ok := m[sel.Selected]; ok {
			return v, true
		}
	}
	return sel.Selected, true
}
