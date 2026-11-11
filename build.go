package goexml

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/five-zombie/goexmlfyne/styles"
)

// Window is a decoded goexml document rendered into Fyne objects.
type Window struct {
	// Title is the window title from the XML.
	Title string
	// Size is the window size from the WxH attribute.
	Size fyne.Size
	// Content is the root Fyne object to show.
	Content fyne.CanvasObject
	// byID maps widget ids to their built objects.
	byID map[string]fyne.CanvasObject
}

// Find returns the built object for an XML id, or nil when absent.
func (w *Window) Find(id string) fyne.CanvasObject {
	return w.byID[id]
}

// SelectValue returns the current choice of a select widget.
// It reports false when id is unknown or not a select.
func (w *Window) SelectValue(id string) (string, bool) {
	o, ok := w.byID[id]
	if !ok {
		return "", false
	}
	s, ok := o.(*widget.Select)
	if !ok {
		return "", false
	}
	return s.Selected, true
}

// Build renders a validated goexml root node into a Fyne window.
// Every onClick name in the document must have a handler in tap.
func Build(root *Node, tap map[string]func()) (*Window, error) {
	if root == nil || root.Type != "goexml" {
		return nil, fmt.Errorf("goexml: root must be a <goexml> node")
	}
	if len(root.Children) != 1 || root.Children[0].Type != "window" {
		return nil, fmt.Errorf("goexml: line %d: <goexml> must hold one <window>", root.Line)
	}
	w := root.Children[0]
	if len(w.Children) == 0 {
		return nil, fmt.Errorf("goexml: line %d: <window> needs content", w.Line)
	}
	sw, sh, err := parseSize(w.Attr("size"), w.Line)
	if err != nil {
		return nil, err
	}
	reg := map[string]fyne.CanvasObject{}
	var objs []fyne.CanvasObject
	for _, c := range w.Children {
		o, err := buildObject(c, tap, reg)
		if err != nil {
			return nil, err
		}
		objs = append(objs, o)
	}
	var content fyne.CanvasObject
	if len(objs) == 1 {
		content = objs[0]
	} else {
		fill := make([]bool, len(w.Children))
		for i, c := range w.Children {
			fill[i] = styleFill(c)
		}
		content = container.New(
			newBoxLayout(true, 0, 4, "start", make([]bool, len(objs)), fill), objs...,
		)
	}
	return &Window{
		Title:   w.Attr("title"),
		Size:    fyne.NewSize(float32(sw), float32(sh)),
		Content: content,
		byID:    reg,
	}, nil
}

// buildObject renders one widget node into a Fyne object and
// records it in reg under its XML id.
func buildObject(n *Node, tap map[string]func(), reg map[string]fyne.CanvasObject) (fyne.CanvasObject, error) {
	o, err := buildWidget(n, tap, reg)
	if err != nil {
		return nil, err
	}
	inner := o
	st := styleOf(n)
	if !st.Pad.IsZero() {
		o = styles.NewPad(st.Pad, o)
		if !inner.Visible() {
			o.Hide()
		}
	}
	if !st.Margin.IsZero() {
		o = styles.NewMargin(st.Margin, o)
	}
	if id := n.Attr("id"); id != "" {
		reg[id] = inner
	}
	return o, nil
}

// buildWidget renders one widget node into a Fyne object.
func buildWidget(n *Node, tap map[string]func(), reg map[string]fyne.CanvasObject) (fyne.CanvasObject, error) {
	switch n.Type {
	case "vbox", "hbox":
		var objs []fyne.CanvasObject
		var expand []bool
		var fill []bool
		for _, c := range n.Children {
			o, err := buildObject(c, tap, reg)
			if err != nil {
				return nil, err
			}
			objs = append(objs, o)
			expand = append(expand, c.Attr("expand") == "true")
			fill = append(fill, styleFill(c))
		}
		box := container.New(
			newBoxLayout(n.Type == "vbox", num(n.Attr("padding")), num(n.Attr("spacing")), alignOf(n), expand, fill),
			objs...,
		)
		return applyState(box, n), nil
	case "border":
		var top, bottom, left, right, center fyne.CanvasObject
		for _, r := range n.Children {
			var objs []fyne.CanvasObject
			for _, g := range r.Children {
				o, err := buildObject(g, tap, reg)
				if err != nil {
					return nil, err
				}
				objs = append(objs, o)
			}
			var o fyne.CanvasObject
			switch len(objs) {
			case 0:
				o = nil
			case 1:
				o = objs[0]
			default:
				fill := make([]bool, len(r.Children))
				for i, g := range r.Children {
					fill[i] = styleFill(g)
				}
				o = container.New(newBoxLayout(true, 0, 4, "start", make([]bool, len(objs)), fill), objs...)
			}
			switch r.Type {
			case "top":
				top = o
			case "bottom":
				bottom = o
			case "left":
				left = o
			case "right":
				right = o
			case "center":
				center = o
			}
		}
		var centers []fyne.CanvasObject
		if center != nil {
			centers = append(centers, center)
		}
		return applyState(container.NewBorder(top, bottom, left, right, centers...), n), nil
	case "label":
		return applyState(widget.NewLabel(n.Attr("text")), n), nil
	case "button":
		name := n.Attr("onClick")
		fn, ok := tap[name]
		if !ok {
			return nil, fmt.Errorf("goexml: line %d: no handler for onClick %q", n.Line, name)
		}
		return applyState(widget.NewButton(n.Attr("text"), fn), n), nil
	case "entry":
		e := widget.NewEntry()
		if text := n.Attr("text"); text != "" {
			e.SetText(text)
		}
		e.PlaceHolder = n.Attr("placeholder")
		return applyState(e, n), nil
	case "select":
		name := n.Attr("onChange")
		fn, ok := tap[name]
		if !ok {
			return nil, fmt.Errorf("goexml: line %d: no handler for onChange %q", n.Line, name)
		}
		var options []string
		selected := ""
		for _, o := range n.Children {
			options = append(options, o.Attr("text"))
			if o.Attr("selected") == "true" {
				selected = o.Attr("text")
			}
		}
		s := widget.NewSelect(options, func(string) { fn() })
		if selected != "" {
			s.SetSelected(selected)
		}
		return applyState(s, n), nil
	case "table":
		var cols []string
		for _, c := range n.Children[0].Children {
			cols = append(cols, c.Attr("text"))
		}
		var rows [][]string
		for _, r := range n.Children[1:] {
			var row []string
			for _, c := range r.Children {
				row = append(row, c.Attr("text"))
			}
			rows = append(rows, row)
		}
		t := widget.NewTable(
			func() (int, int) { return len(rows), len(cols) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(id widget.TableCellID, cell fyne.CanvasObject) {
				cell.(*widget.Label).SetText(rows[id.Row][id.Col])
			},
		)
		t.ShowHeaderRow = true
		t.CreateHeader = func() fyne.CanvasObject { return widget.NewLabel("") }
		t.UpdateHeader = func(id widget.TableCellID, cell fyne.CanvasObject) {
			cell.(*widget.Label).SetText(cols[id.Col])
		}
		if name := n.Attr("onSelect"); name != "" {
			fn, ok := tap[name]
			if !ok {
				return nil, fmt.Errorf("goexml: line %d: no handler for onSelect %q", n.Line, name)
			}
			t.OnSelected = func(widget.TableCellID) { fn() }
		}
		return applyState(t, n), nil
	default:
		return nil, fmt.Errorf("goexml: line %d: cannot build <%s>", n.Line, n.Type)
	}
}

// styleOf reads the style attribute leniently for the builder.
// Decode is the strict gate; invalid styles become the zero Style here.
func styleOf(n *Node) styles.Style {
	st, _ := styles.ParseStyle(n.Attr("style"))
	return st
}

// styleFill reports whether the node fills the parent cross axis.
func styleFill(n *Node) bool {
	return styleOf(n).Fill
}

// applyState enforces the visible and disabled attributes.
func applyState(o fyne.CanvasObject, n *Node) fyne.CanvasObject {
	if n.Attr("visible") == "false" {
		o.Hide()
	}
	if n.Attr("disabled") == "true" {
		if d, ok := o.(fyne.Disableable); ok {
			d.Disable()
		}
	}
	return o
}

// alignOf returns the cross-axis alignment, defaulting to start.
func alignOf(n *Node) string {
	if a := n.Attr("align"); a != "" {
		return a
	}
	return "start"
}

// num parses a non-negative integer attribute, defaulting to zero.
func num(s string) float32 {
	i, _ := strconv.Atoi(s)
	return float32(i)
}
