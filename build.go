package goexml

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
	var hints []childHint
	for _, c := range w.Children {
		o, err := buildObject(c, tap, reg)
		if err != nil {
			return nil, err
		}
		objs = append(objs, o)
		hints = append(hints, hintOf(c))
	}
	var content fyne.CanvasObject
	if len(objs) == 1 {
		content = objs[0]
	} else {
		content = container.New(
			newBoxLayout(true, 0, 4, "start", "start", hints), objs...,
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
	if n.Type == "table" {
		// Tables are wrapped in a sizing box; register the table itself
		// so Find returns *widget.Table, like every other widget.
		for {
			c, ok := inner.(*fyne.Container)
			if !ok || len(c.Objects) != 1 {
				break
			}
			inner = c.Objects[0]
		}
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
		vertical := n.Type == "vbox"
		var objs []fyne.CanvasObject
		var hints []childHint
		for _, c := range n.Children {
			o, err := buildObject(c, tap, reg)
			if err != nil {
				return nil, err
			}
			objs = append(objs, o)
			hints = append(hints, hintOf(c))
		}
		var box fyne.CanvasObject
		if n.Type == "vbox" && n.Attr("column") == "grid" {
			box = container.New(
				newGridLayout(num(n.Attr("padding")), num(n.Attr("spacing")), gridMin(n), gridMax(n), hints),
				objs...,
			)
		} else if n.Type == "vbox" && n.Attr("column") != "" {
			return nil, fmt.Errorf("goexml: line %d: unknown column mode %q, want grid", n.Line, n.Attr("column"))
		} else {
			box = container.New(
				newBoxLayout(vertical, num(n.Attr("padding")), num(n.Attr("spacing")), alignOf(n), justifyOf(n), hints),
				objs...,
			)
		}
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
				var hints []childHint
				for _, g := range r.Children {
					hints = append(hints, hintOf(g))
				}
				o = container.New(newBoxLayout(true, 0, 4, "start", "start", hints), objs...)
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
	case "text":
		body := strings.TrimSpace(n.Attr("v"))
		if body == "" {
			body = n.Text
		}
		l := widget.NewLabel(body)
		// A text block wraps by default; "off" keeps one line.
		switch n.Attr("wrap") {
		case "off":
		case "break":
			l.Wrapping = fyne.TextWrapBreak
		default:
			l.Wrapping = fyne.TextWrapWord
		}
		return applyState(l, n), nil
	case "button":
		name := n.Attr("onClick")
		fn, ok := tap[name]
		if !ok {
			return nil, fmt.Errorf("goexml: line %d: no handler for onClick %q", n.Line, name)
		}
		return applyState(widget.NewButton(n.Attr("text"), fn), n), nil
	case "entry":
		e := widget.NewEntry()
		// Direct assignment: SetText measures text, which needs a
		// running app. The renderer reads Text when shown.
		e.Text = n.Attr("text")
		e.PlaceHolder = n.Attr("placeholder")
		return applyState(e, n), nil
	case "rect":
		// Decode guarantees a valid color; invalid input here can only
		// come from hand-built trees, so fall back to opaque black.
		r, g, b, _ := parseHexColor(n.Attr("color"), n.Line)
		return applyState(canvas.NewRectangle(color.NRGBA{R: r, G: g, B: b, A: 255}), n), nil
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
			// Direct assignment: SetSelected refreshes, which needs a
			// running app, and it would also fire OnChanged during build.
			// The renderer picks up Selected when the widget is shown.
			s.Selected = selected
		}
		return applyState(s, n), nil
	case "table":
		var cols []string
		for _, c := range n.Children[0].Children {
			cols = append(cols, cellText(c))
		}
		var rows [][]string
		for _, r := range n.Children[1:] {
			var row []string
			for _, c := range r.Children {
				row = append(row, cellText(c))
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
		widths := columnWidths(cols, rows)
		for i, w := range widths {
			t.SetColumnWidth(i, w)
		}
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
		applyState(t, n)
		var totalW float32
		for _, w := range widths {
			totalW += w
		}
		// The template MinSize only fits about one row; size the
		// wrapper for the header plus every data row so real
		// content is visible.
		rowH := float32(0)
		probe := func(s string) {
			if h := widget.NewLabel(s).MinSize().Height; h > rowH {
				rowH = h
			}
		}
		for _, c := range cols {
			probe(c)
		}
		for _, r := range rows {
			for _, s := range r {
				probe(s)
			}
		}
		box := container.New(newTableLayout(fyne.NewSize(totalW, rowH*float32(len(rows)+1)+8)), t)
		if !t.Visible() {
			box.Hide()
		}
		return box, nil
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

// cellText returns a cell's text: the text attribute, or the inner
// text when the attribute is absent. Validation guarantees exactly
// one of the two is present.
func cellText(n *Node) string {
	if t := strings.TrimSpace(n.Attr("text")); t != "" {
		return t
	}
	return n.Text
}

// columnWidths returns one width per column: the widest cell content
// plus padding, so real text is never truncated to fit.
func columnWidths(cols []string, rows [][]string) []float32 {
	measure := func(s string) float32 { return widget.NewLabel(s).MinSize().Width }
	widths := make([]float32, len(cols))
	for i, c := range cols {
		widths[i] = measure(c)
	}
	for _, r := range rows {
		for i := range r {
			if i < len(widths) {
				if w := measure(r[i]); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}
	for i := range widths {
		widths[i] += 16
	}
	return widths
}

// hintOf maps a validated node to its layout hint.
// expand="true" is shorthand for w-full plus h-full.
func hintOf(n *Node) childHint {
	st := styleOf(n)
	expand := n.Attr("expand") == "true"
	w := strings.TrimSpace(n.Attr("wrap"))
	return childHint{
		wFull:  st.WFull || expand,
		hFull:  st.HFull || expand,
		fixedH: st.FixedH,
		wrap:   w == "word" || w == "break" || (n.Type == "text" && w != "off"),
	}
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

// justifyOf returns the main-axis distribution, defaulting to start.
func justifyOf(n *Node) string {
	if j := n.Attr("justify"); j != "" {
		return j
	}
	return "start"
}

// num parses a non-negative integer attribute, defaulting to zero.
func num(s string) float32 {
	i, _ := strconv.Atoi(s)
	return float32(i)
}

// gridMin returns the minimum grid column count, defaulting to 1.
// Decode is the strict gate; garbage falls back to the default here.
func gridMin(n *Node) int {
	if v, err := strconv.Atoi(n.Attr("min")); err == nil && v >= 1 {
		return v
	}
	return 1
}

// gridMax returns the maximum grid column count, or 0 for no upper clamp.
func gridMax(n *Node) int {
	if v, err := strconv.Atoi(n.Attr("max")); err == nil && v >= 1 {
		return v
	}
	return 0
}
