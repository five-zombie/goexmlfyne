package goexml

import (
	"bytes"
	"fmt"
	"image"
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
	// resources is the resource set the build resolved image src keys
	// against; SetImage re-resolves against it.
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
	// docIDs are the document's own ids: fragments must not collide
	// with them, while fragment ids may replace each other across
	// rebuilds (last one wins).
	docIDs map[string]bool
	// itemIDs are the ids contributed by the most recently built
	// fragment. A rebuild retires the ids the new fragment no longer
	// declares, so widgets, selects and templates of an old page never
	// outlive it in Find/Text/SetValue.
	itemIDs map[string]bool
	// tap is the handler map Build received, reused by BuildItem.
	tap map[string]func()
}

// buildCtx carries the shared state of one build pass.
type buildCtx struct {
	tap          map[string]func()
	resources    map[string]fyne.Resource
	reg          map[string]fyne.CanvasObject
	nodes        map[string]*Node
	containers   map[string]*Container
	templates    map[string]string
	selectValues map[string]map[string]string
	optionRaw    map[string][]optPair
	slot         *fyne.Container // slot holder of the enclosing container
}

// optPair is one select option: its raw (possibly templated) label and
// its value.
type optPair struct{ raw, value string }

// Find returns the built object for an XML id, or nil when absent.
func (w *Window) Find(id string) fyne.CanvasObject {
	return w.byID[id]
}

// SelectValue returns the option value of the current choice of a
// select widget. It reports false when id is unknown or not a select.
// A choice set outside the declared options falls back to its label.
func (w *Window) SelectValue(id string) (string, bool) {
	o, ok := w.byID[id]
	if !ok {
		return "", false
	}
	s, ok := o.(*widget.Select)
	if !ok {
		return "", false
	}
	if m, ok := w.selectValues[id]; ok {
		if v, ok := m[s.Selected]; ok {
			return v, true
		}
	}
	return s.Selected, true
}

// Build renders a validated goexml root node into a Fyne window.
// Every onClick name in the document must have a handler in tap. Every
// <container> in the document must get its slot content from contents,
// keyed by the container id; an unfilled slot or an unknown id is an
// error.
// Build turns a decoded <goexml> tree into a Window. tap holds the
// handler named by onClick/onChange/onSelect, contents fills every
// container's slot keyed by container id, and resources holds the
// named image payloads that <image src> resolves against; a missing
// handler, container or resource key fails the build.
func Build(root *Node, tap map[string]func(), contents map[string]fyne.CanvasObject, resources map[string]fyne.Resource) (*Window, error) {
	if root == nil || root.Type != "goexml" {
		return nil, fmt.Errorf("goexml: root must be a <goexml> node")
	}
	if len(root.Children) != 1 || root.Children[0].Type != "window" {
		return nil, nodeErr(root, "must hold one <window>")
	}
	w := root.Children[0]
	if len(w.Children) == 0 {
		return nil, nodeErr(w, "needs content")
	}
	sw, sh, err := parseSize(w, w.Attr("size"))
	if err != nil {
		return nil, err
	}
	ctx := &buildCtx{
		tap:          tap,
		resources:    resources,
		reg:          map[string]fyne.CanvasObject{},
		nodes:        map[string]*Node{},
		containers:   map[string]*Container{},
		templates:    map[string]string{},
		selectValues: map[string]map[string]string{},
		optionRaw:    map[string][]optPair{},
	}
	registerSelects(w, ctx)
	var objs []fyne.CanvasObject
	var hints []childHint
	for _, c := range w.Children {
		o, err := buildObject(c, ctx)
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
	win := &Window{
		Title:        w.Attr("title"),
		Size:         fyne.NewSize(float32(sw), float32(sh)),
		Content:      content,
		resources:    resources,
		byID:         ctx.reg,
		nodes:        ctx.nodes,
		containers:   ctx.containers,
		templates:    ctx.templates,
		selectValues: ctx.selectValues,
		optionRaw:    ctx.optionRaw,
		docIDs:       map[string]bool{},
		itemIDs:      map[string]bool{},
		tap:          tap,
	}
	var idWalk func(n *Node)
	idWalk = func(n *Node) {
		if id := n.Attr("id"); id != "" {
			win.docIDs[id] = true
		}
		for _, c := range n.Children {
			idWalk(c)
		}
	}
	idWalk(root)
	// Every container must be filled through the contents map — an
	// unfilled slot would render as an invisible hole; an unknown key
	// is almost certainly a typo in the Go side.
	for id := range contents {
		if _, ok := win.containers[id]; !ok {
			return nil, fmt.Errorf("goexml: no container with id %q", id)
		}
	}
	for id, c := range win.containers {
		obj, ok := contents[id]
		if !ok {
			return nil, nodeErr(c.node, "slot must be filled")
		}
		c.SetContent(obj)
	}
	return win, nil
}

// BuildItem builds a decoded goexml-item fragment into a Fyne object,
// registering its ids into this window's global id space. The fragment
// must not collide with the document's own ids; ids of previously built
// fragments are replaced, so pages can be rebuilt. Every container in
// the fragment must get its content from contents, keyed by id.
func (w *Window) BuildItem(node *Node, contents map[string]fyne.CanvasObject, resources map[string]fyne.Resource) (fyne.CanvasObject, error) {
	if node == nil || node.Type != "goexml-item" {
		return nil, fmt.Errorf("goexml: node must be a <goexml-item> root")
	}
	if len(node.Children) != 1 {
		return nil, nodeErr(node, "must hold exactly one child")
	}
	var itemContainers []string
	fragIDs := map[string]bool{}
	var pre func(n *Node) error
	pre = func(n *Node) error {
		if id := n.Attr("id"); id != "" {
			if w.docIDs[id] {
				return nodeErr(n, "id %q is already used by the document", id)
			}
			fragIDs[id] = true
			if n.Type == "container" {
				itemContainers = append(itemContainers, id)
			}
		}
		for _, c := range n.Children {
			if err := pre(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := pre(node.Children[0]); err != nil {
		return nil, err
	}
	// Last call wins: SetImage after this build resolves against the
	// newest resource set.
	w.resources = resources
	ctx := &buildCtx{
		tap:          w.tap,
		resources:    resources,
		reg:          map[string]fyne.CanvasObject{},
		nodes:        map[string]*Node{},
		containers:   map[string]*Container{},
		templates:    map[string]string{},
		selectValues: map[string]map[string]string{},
		optionRaw:    map[string][]optPair{},
	}
	// The fragment's selects join the window's value tables: without
	// this walk SelectValue would silently return labels instead of
	// values, and SetValue/SetValueFormat would reject valid options.
	registerSelects(node.Children[0], ctx)
	obj, err := buildObject(node.Children[0], ctx)
	if err != nil {
		return nil, err
	}
	for id := range contents {
		if !slicesContain(itemContainers, id) {
			return nil, fmt.Errorf("goexml: no container with id %q in the fragment", id)
		}
	}
	for _, id := range itemContainers {
		obj, ok := contents[id]
		if !ok {
			return nil, nodeErr(ctx.containers[id].node, "slot must be filled")
		}
		ctx.containers[id].SetContent(obj)
	}
	// The fragment built clean: retire the ids the previous fragment
	// contributed that this fragment no longer declares — stale
	// optionRaw tables, templates and orphan widgets never outlive
	// their page. Ids both fragments share are re-merged below (last
	// wins), keeping the rebuild-per-switch lifecycle.
	for id := range w.itemIDs {
		if !fragIDs[id] {
			delete(w.byID, id)
			delete(w.nodes, id)
			delete(w.containers, id)
			delete(w.templates, id)
			delete(w.selectValues, id)
			delete(w.optionRaw, id)
		}
	}
	w.itemIDs = fragIDs
	// Shared ids are not covered by the retirement above: a select or
	// text that rebuilt plain must not inherit the previous page's
	// optionRaw or template, so drop the per-id state the new fragment
	// does not declare.
	for id := range ctx.nodes {
		if _, ok := ctx.templates[id]; !ok {
			delete(w.templates, id)
		}
	}
	for id := range ctx.selectValues {
		if _, ok := ctx.optionRaw[id]; !ok {
			delete(w.optionRaw, id)
		}
	}
	// Merge the new fragment's registrations into the window. Building
	// went into temporary maps, so a failed BuildItem leaves no orphan
	// ids behind — errors are side-effect free.
	for id, o := range ctx.reg {
		w.byID[id] = o
	}
	for id, n := range ctx.nodes {
		w.nodes[id] = n
	}
	for id, c := range ctx.containers {
		w.containers[id] = c
	}
	for id, t := range ctx.templates {
		w.templates[id] = t
	}
	for id, m := range ctx.selectValues {
		w.selectValues[id] = m
	}
	for id, p := range ctx.optionRaw {
		w.optionRaw[id] = p
	}
	return obj, nil
}

// slicesContain reports whether s holds id.
func slicesContain(s []string, id string) bool {
	for _, v := range s {
		if v == id {
			return true
		}
	}
	return false
}

// registerSelects pre-walks a tree and fills ctx.selectValues and
// ctx.optionRaw for every select, so SelectValue, SetValue and
// SetValueFormat work on any build path — Build and BuildItem alike.
// The select's value table lives here, not in buildWidget: the option
// texts a fragment declares must join the document's id space the same
// way a document's own options do.
func registerSelects(n *Node, ctx *buildCtx) {
	if n.Type == "select" {
		m := map[string]string{}
		var pairs []optPair
		templated := false
		for _, o := range n.Children {
			m[o.Attr("text")] = o.Attr("value")
			pairs = append(pairs, optPair{raw: o.Attr("text"), value: o.Attr("value")})
			if strings.Contains(o.Attr("text"), "{{") {
				templated = true
			}
		}
		ctx.selectValues[n.Attr("id")] = m
		if templated {
			ctx.optionRaw[n.Attr("id")] = pairs
		}
	}
	for _, c := range n.Children {
		registerSelects(c, ctx)
	}
}

// buildObject renders one widget node into a Fyne object and
// records it in ctx.reg under its XML id.
func buildObject(n *Node, ctx *buildCtx) (fyne.CanvasObject, error) {
	o, err := buildWidget(n, ctx)
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
	// The visual layer (bg/border/rounded) wraps the padded content:
	// the background covers the whole box including its padding.
	if st.Vis.Active() {
		o = styles.NewVisual(st.Vis, o)
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
		ctx.reg[id] = inner
		ctx.nodes[id] = n
		switch n.Type {
		case "label", "button", "entry":
			if strings.Contains(n.Attr("text"), "{{") {
				ctx.templates[id] = n.Attr("text")
			}
		case "text":
			if strings.Contains(n.Attr("v"), "{{") {
				ctx.templates[id] = n.Attr("v")
			}
		}
	}
	return o, nil
}

// buildWidget renders one widget node into a Fyne object.
func buildWidget(n *Node, ctx *buildCtx) (fyne.CanvasObject, error) {
	switch n.Type {
	case "slot":
		// The slot placeholder: zero size until SetContent fills it.
		// Decode guarantees one slot per container; this guard keeps
		// hand-built trees honest.
		if ctx.slot != nil {
			return nil, nodeErr(n, "duplicate slot in container")
		}
		holder := container.New(&holderLayout{})
		ctx.slot = holder
		return holder, nil
	case "container":
		if len(n.Children) != 1 {
			return nil, nodeErr(n, "needs exactly one child")
		}
		prev := ctx.slot
		ctx.slot = nil
		obj, err := buildObject(n.Children[0], ctx)
		slot := ctx.slot
		ctx.slot = prev
		if err != nil {
			return nil, err
		}
		if slot == nil {
			return nil, nodeErr(n, "needs exactly one <slot>")
		}
		c := &Container{
			Container: container.New(&holderLayout{}, obj),
			node:      n,
			slot:      slot,
		}
		ctx.containers[n.Attr("id")] = c
		// Container is a widget like any other: visible, style padding
		// and margin must not be accepted-then-dead attributes.
		return applyState(c, n), nil
	case "vbox", "hbox":
		vertical := n.Type == "vbox"
		var objs []fyne.CanvasObject
		var hints []childHint
		for _, c := range n.Children {
			o, err := buildObject(c, ctx)
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
			return nil, nodeErr(n, "unknown column mode %q, want grid", n.Attr("column"))
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
				o, err := buildObject(g, ctx)
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
		fn, ok := ctx.tap[name]
		if !ok {
			return nil, nodeErr(n, "no handler for onClick %q", name)
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
		r, g, b, _ := parseHexColor(n, n.Attr("color"))
		return applyState(canvas.NewRectangle(color.NRGBA{R: r, G: g, B: b, A: 255}), n), nil
	case "image":
		res, ok := ctx.resources[n.Attr("src")]
		if !ok {
			return nil, nodeErr(n, "unknown image resource %q", n.Attr("src"))
		}
		// A resource that exists but is not a decodable image would
		// render as an empty box and only log at paint time; fail the
		// build instead, like any other broken reference. The config
		// also gives the intrinsic pixel size for the min size.
		cfg, _, err := image.DecodeConfig(bytes.NewReader(res.Content()))
		if err != nil {
			return nil, nodeErr(n, "bad image resource %q: %v", n.Attr("src"), err)
		}
		img := canvas.NewImageFromResource(res)
		// The backend gives images no intrinsic min size in the fill
		// modes (they scale to the allocated box, min 1x1), so set it
		// explicitly: without style utilities an image is as large as
		// its pixels, one px = one coordinate unit like everywhere else.
		img.SetMinSize(fyne.NewSize(float32(cfg.Width), float32(cfg.Height)))
		// Decode guarantees one of the three values ("" = contain).
		switch n.Attr("fit") {
		case "cover":
			img.FillMode = canvas.ImageFillCover
		case "fill":
			img.FillMode = canvas.ImageFillStretch
		default:
			img.FillMode = canvas.ImageFillContain
		}
		return applyState(img, n), nil
	case "select":
		name := n.Attr("onChange")
		fn, ok := ctx.tap[name]
		if !ok {
			return nil, nodeErr(n, "no handler for onChange %q", name)
		}
		var options []string
		selected := ""
		for _, o := range n.Children {
			options = append(options, o.Attr("text"))
			if _, ok := o.Attrs["selected"]; ok {
				// Validation rejects a valued selected; an empty
				// selected="" marks the chosen option.
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
		if len(n.Children) == 0 {
			// Hand-built trees skip Decode validation; reject here
			// instead of panicking on n.Children[0] below.
			return nil, nodeErr(n, "needs at least one <row>")
		}
		// Same contract for ragged rows: uneven cell counts would panic
		// later in the UpdateCell callback during render.
		want := len(n.Children[0].Children)
		for _, r := range n.Children[1:] {
			if len(r.Children) != want {
				return nil, nodeErr(r, "has %d cells, want %d", len(r.Children), want)
			}
		}
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
		widths, rowH := tableMetrics(cols, rows)
		for i, w := range widths {
			t.SetColumnWidth(i, w)
		}
		t.ShowHeaderRow = true
		t.CreateHeader = func() fyne.CanvasObject { return widget.NewLabel("") }
		t.UpdateHeader = func(id widget.TableCellID, cell fyne.CanvasObject) {
			cell.(*widget.Label).SetText(cols[id.Col])
		}
		if name := n.Attr("onSelect"); name != "" {
			fn, ok := ctx.tap[name]
			if !ok {
				return nil, nodeErr(n, "no handler for onSelect %q", name)
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
		box := container.New(newTableLayout(fyne.NewSize(totalW, rowH*float32(len(rows)+1)+8)), t)
		if !t.Visible() {
			box.Hide()
		}
		return box, nil
	default:
		return nil, nodeErr(n, "cannot build")
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

// tableMetrics measures the table in a single pass: column widths and
// the shared row height come from one walk over the header and cells,
// and each distinct string is measured only once — duplicate content
// (status columns, numbers) costs one map lookup, not two labels.
func tableMetrics(cols []string, rows [][]string) (widths []float32, rowH float32) {
	cache := map[string]fyne.Size{}
	measure := func(s string) fyne.Size {
		if sz, ok := cache[s]; ok {
			return sz
		}
		sz := widget.NewLabel(s).MinSize()
		cache[s] = sz
		return sz
	}
	widths = make([]float32, len(cols))
	for i, c := range cols {
		sz := measure(c)
		widths[i] = sz.Width
		if sz.Height > rowH {
			rowH = sz.Height
		}
	}
	for _, r := range rows {
		for i, s := range r {
			sz := measure(s)
			if i < len(widths) && sz.Width > widths[i] {
				widths[i] = sz.Width
			}
			if sz.Height > rowH {
				rowH = sz.Height
			}
		}
	}
	// Plus padding, so real text is never truncated to fit.
	for i := range widths {
		widths[i] += 16
	}
	return widths, rowH
}

// hintOf maps a validated node to its layout hint.
func hintOf(n *Node) childHint {
	st := styleOf(n)
	w := strings.TrimSpace(n.Attr("wrap"))
	// The standard defines h-N as the exact content height regardless of
	// the widget's content. The pad and margin wrappers around the widget
	// also take a share of the height the box layout assigns, so the hint
	// carries the wrapper total to keep the content itself at N*4 px.
	// h-0 is a real zero height, so it takes the same treatment.
	fixedH := st.FixedH
	fixedSet := st.HasFixedH()
	if fixedSet {
		fixedH += st.Pad.Top + st.Pad.Bottom + st.Margin.Top + st.Margin.Bottom
	}
	return childHint{
		wFull:    st.WFull,
		hFull:    st.HFull,
		fixedH:   fixedH,
		fixedSet: fixedSet,
		wrap:     w == "word" || w == "break" || (n.Type == "text" && w != "off"),
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
