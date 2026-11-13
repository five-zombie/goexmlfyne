package goexml

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestMain(m *testing.M) {
	a := test.NewApp()
	code := m.Run()
	a.Quit()
	os.Exit(code)
}

func buildValid(t *testing.T, tap map[string]func()) *Window {
	t.Helper()
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, tap, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	return w
}

func TestBuildForm(t *testing.T) {
	saved := false
	w := buildValid(t, map[string]func(){
		"OnCancel": func() {},
		"OnSave":   func() { saved = true },
	})
	if w.Title != "Form" {
		t.Fatalf("title = %q, want Form", w.Title)
	}
	if w.Size != fyne.NewSize(800, 600) {
		t.Fatalf("size = %v, want 800x600", w.Size)
	}
	box, ok := w.Content.(*fyne.Container)
	if !ok {
		t.Fatalf("content is %T, want *fyne.Container", w.Content)
	}
	if len(box.Objects) != 3 {
		t.Fatalf("content children = %d, want 3", len(box.Objects))
	}
	actions, ok := box.Objects[2].(*fyne.Container)
	if !ok || len(actions.Objects) != 2 {
		t.Fatalf("actions row is %T with %d children, want container with 2",
			box.Objects[2], len(actions.Objects))
	}
	btn, ok := actions.Objects[1].(*widget.Button)
	if !ok {
		t.Fatalf("save control is %T, want *widget.Button", actions.Objects[1])
	}
	btn.OnTapped()
	if !saved {
		t.Fatal("tapping save did not call OnSave")
	}
}

func TestBuildUnknownHandler(t *testing.T) {
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	_, err = Build(root, map[string]func(){}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no handler") {
		t.Fatalf("expected no-handler error, got %v", err)
	}
}

func TestBuildState(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><label id="l" text="x" visible="false"/>` +
		`<entry id="e" disabled="true"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	if box.Objects[0].Visible() {
		t.Fatal("hidden label is visible")
	}
	entry, ok := box.Objects[1].(*widget.Entry)
	if !ok {
		t.Fatalf("second child is %T, want *widget.Entry", box.Objects[1])
	}
	if !entry.Disabled() {
		t.Fatal("disabled entry reports enabled")
	}
}

func TestBuildBorder(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<border id="b"><top><label id="t" text="top"/></top>` +
		`<center><label id="c" text="mid"/></center></border></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if _, ok := w.Content.(*fyne.Container); !ok {
		t.Fatalf("content is %T, want *fyne.Container", w.Content)
	}
}

func TestBuildSelectTable(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><select id="s" onChange="OnPick">` +
		`<option value="a" text="A"/><option value="b" text="B" selected=""/></select>` +
		`<table id="t"><row><cell text="Name"/><cell text="Age"/></row>` +
		`<row><cell text="Ann"/><cell text="30"/></row>` +
		`<row><cell text="Bo"/><cell text="25"/></row></table>` +
		`</vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	picked := ""
	_, err = Build(root, map[string]func(){}, nil, nil)
	if err == nil {
		t.Fatal("expected no-handler error for missing OnPick, got nil")
	}
	pick := func() { picked = "yes" }
	w, err := Build(root, map[string]func(){"OnPick": pick}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	sel, ok := box.Objects[0].(*widget.Select)
	if !ok {
		t.Fatalf("first child is %T, want *widget.Select", box.Objects[0])
	}
	if len(sel.Options) != 2 || sel.Options[0] != "A" || sel.Selected != "B" {
		t.Fatalf("select options=%v selected=%q, want [A B] and B", sel.Options, sel.Selected)
	}
	sel.OnChanged("a")
	if picked != "yes" {
		t.Fatal("changing select did not call OnPick")
	}
	wrap, ok := box.Objects[1].(*fyne.Container)
	if !ok {
		t.Fatalf("second child is %T, want table wrapper *fyne.Container", box.Objects[1])
	}
	tab, ok := wrap.Objects[0].(*widget.Table)
	if !ok {
		t.Fatalf("wrapped object is %T, want *widget.Table", wrap.Objects[0])
	}
	if rows, cols := tab.Length(); rows != 2 || cols != 2 {
		t.Fatalf("table size = %dx%d, want 2x2", rows, cols)
	}
	if !tab.ShowHeaderRow || tab.UpdateHeader == nil {
		t.Fatal("table header row is not enabled")
	}
	header := tab.CreateHeader().(*widget.Label)
	tab.UpdateHeader(widget.TableCellID{Row: -1, Col: 1}, header)
	if header.Text != "Age" {
		t.Fatalf("header[1] = %q, want Age", header.Text)
	}
}

func TestBuildTableSelect(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<table id="t" onSelect="OnRow"><row><cell text="A"/><cell text="B"/></row>` +
		`<row><cell text="1"/><cell text="2"/></row></table></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if _, err := Build(root, map[string]func(){}, nil, nil); err == nil {
		t.Fatal("expected no-handler error for missing OnRow, got nil")
	}
	chosen := false
	w, err := Build(root, map[string]func(){"OnRow": func() { chosen = true }}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	wrap, ok := w.Content.(*fyne.Container)
	if !ok {
		t.Fatalf("content is %T, want table wrapper *fyne.Container", w.Content)
	}
	tab, ok := wrap.Objects[0].(*widget.Table)
	if !ok {
		t.Fatalf("wrapped object is %T, want *widget.Table", wrap.Objects[0])
	}
	if wrap.MinSize().Height <= tab.MinSize().Height {
		t.Fatalf("wrapper height %v does not cover header plus rows (template %v)", wrap.MinSize().Height, tab.MinSize().Height)
	}
	if tab.OnSelected == nil {
		t.Fatal("table has no OnSelected handler")
	}
	tab.OnSelected(widget.TableCellID{Row: 0, Col: 1})
	if !chosen {
		t.Fatal("selecting a cell did not call OnRow")
	}
}

func TestBuildTableNoRows(t *testing.T) {
	root := &Node{Type: "goexml", Attrs: map[string]string{"version": "0.1.0"}, Children: []*Node{
		{Type: "window", Attrs: map[string]string{"title": "x", "size": "10x10"}, Children: []*Node{
			{Type: "table", Line: 3},
		}},
	}}
	_, err := Build(root, map[string]func(){}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "needs at least one <row>") {
		t.Fatalf("Build error = %v, want table row complaint", err)
	}
}

func TestBuildTableRaggedCells(t *testing.T) {
	root := &Node{Type: "goexml", Attrs: map[string]string{"version": "0.1.0"}, Children: []*Node{
		{Type: "window", Attrs: map[string]string{"title": "x", "size": "10x10"}, Children: []*Node{
			{Type: "table", Attrs: map[string]string{"id": "t"}, Children: []*Node{
				{Type: "row", Children: []*Node{
					{Type: "cell", Attrs: map[string]string{"text": "A"}},
					{Type: "cell", Attrs: map[string]string{"text": "B"}},
				}},
				{Type: "row", Line: 6, Children: []*Node{
					{Type: "cell", Attrs: map[string]string{"text": "1"}},
				}},
			}},
		}},
	}}
	_, err := Build(root, map[string]func(){}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "has 1 cells, want 2") {
		t.Fatalf("Build error = %v, want ragged row complaint", err)
	}
}

// h-N sizes the content; the pad and margin wrappers take their share
// from the height the layout assigns, never from the content.
// A child carrying w-full and h-full takes the parent's full content
// area: the fill dimensions resolve against the parent, not siblings.
func TestBuildFillTakesParentArea(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x100">` +
		`<vbox id="b"><rect id="r" color="#2e2e2e" style="w-full h-full"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(200, 100))
	r := box.Objects[0]
	if got := r.Size(); got.Width != 200 || got.Height != 100 {
		t.Fatalf("w-full h-full child = %vx%v, want 200x100 (parent area)", got.Width, got.Height)
	}
}

// Fill dimensions resolve against the parent's content area: a
// padded parent keeps its padding, the child never overlaps it.
func TestBuildFillRespectsParentPadding(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x100">` +
		`<vbox id="b" padding="8"><rect id="r" color="#2e2e2e" style="w-full h-full"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(200, 100))
	r := box.Objects[0]
	if got := r.Size(); got.Width != 184 || got.Height != 84 {
		t.Fatalf("w-full h-full child = %vx%v, want 184x84 (200x100 minus padding 8)", got.Width, got.Height)
	}
	if got := r.Position(); got.X != 8 || got.Y != 8 {
		t.Fatalf("w-full h-full child at (%v,%v), want (8,8)", got.X, got.Y)
	}
}

// h-0 is a real zero height: an invisible element that still takes
// part in layout through its margin (a spacer).
func TestBuildFixedHeightZeroSpacer(t *testing.T) {
	for _, tc := range []struct {
		style string
		outer float32
	}{
		{"h-0", 0},      // invisible, no footprint
		{"h-0 m-2", 16}, // invisible, pushes neighbours apart via margin
	} {
		doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
			`<vbox id="b"><rect id="r" color="#2e2e2e" style="` + tc.style + `"/></vbox></window></goexml>`
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode %s returned error: %v", tc.style, err)
		}
		w, err := Build(root, map[string]func(){}, nil, nil)
		if err != nil {
			t.Fatalf("Build %s returned error: %v", tc.style, err)
		}
		box := w.Content.(*fyne.Container)
		box.Layout.Layout(box.Objects, fyne.NewSize(300, 400))
		outer := box.Objects[0]
		if outer.Size().Height != tc.outer {
			t.Fatalf("%s wrapper height = %v, want %v", tc.style, outer.Size().Height, tc.outer)
		}
	}
}

func TestBuildFixedHeightWrappers(t *testing.T) {
	for _, tc := range []struct {
		style string
		outer float32 // wrapper height the box layout assigns
	}{
		{"h-2 p-2", 24},     // 8 content + 8+8 padding
		{"h-2 m-2", 24},     // 8 content + 8+8 margin
		{"h-2 p-2 m-2", 40}, // 8 content + 16 padding + 16 margin
	} {
		doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
			`<vbox id="b"><rect id="r" color="#2e2e2e" style="` + tc.style + `"/></vbox></window></goexml>`
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode %s returned error: %v", tc.style, err)
		}
		w, err := Build(root, map[string]func(){}, nil, nil)
		if err != nil {
			t.Fatalf("Build %s returned error: %v", tc.style, err)
		}
		box := w.Content.(*fyne.Container)
		box.Layout.Layout(box.Objects, fyne.NewSize(300, 400))
		outer := box.Objects[0]
		if outer.Size().Height != tc.outer {
			t.Fatalf("%s wrapper height = %v, want %v", tc.style, outer.Size().Height, tc.outer)
		}
		inner := fyne.CanvasObject(outer)
		for {
			c, ok := inner.(*fyne.Container)
			if !ok || len(c.Objects) != 1 {
				break
			}
			inner = c.Objects[0]
		}
		if inner.Size().Height != 8 {
			t.Fatalf("%s content height = %v, want 8", tc.style, inner.Size().Height)
		}
	}
}

func TestScrollWrapsTallContent(t *testing.T) {
	data, err := os.ReadFile("examples/showcase.xml")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	root, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){
		"OnCancel": func() {}, "OnSave": func() {}, "OnPick": func() {},
		"OnRow": func() {}, "OnDemo": func() {},
	}, nil, map[string]fyne.Resource{
		"assets/demo.png": fyne.NewStaticResource("demo.png", pngBytes(2, 1, color.NRGBA{A: 255})),
	})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	// Lay out once first: wrapped rows only reach their true height
	// after layout, so the fresh minimum understates tall content.
	w.Content.Resize(w.Size)
	full := w.Content.MinSize()
	if full.Height <= w.Size.Height {
		t.Fatalf("content height %v fits window %v, test proves nothing", full.Height, w.Size.Height)
	}
	// The demo shows content through a scroll container; the scroll
	// must report a small size even for huge content, otherwise Fyne
	// clamps the window content. The driver-level clamp itself needs
	// a display and is not reproducible here.
	view := container.NewScroll(w.Content)
	if got := view.MinSize(); got.Height >= full.Height || got.Height > w.Size.Height {
		t.Fatalf("scroll size %v does not fit tall content %v in window %v", got, full, w.Size)
	}
}

func TestWindowFindSelectValue(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><select id="city" onChange="OnPick">` +
		`<option value="jkt" text="Jakarta"/><option value="bdg" text="Bandung"/></select>` +
		`<entry id="nama"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){"OnPick": func() {}}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if w.Find("hilang") != nil {
		t.Fatal("Find of unknown id did not return nil")
	}
	if _, ok := w.Find("b").(*fyne.Container); !ok {
		t.Fatalf("Find(b) is %T, want *fyne.Container", w.Find("b"))
	}
	got, ok := w.SelectValue("city")
	if !ok || got != "" {
		t.Fatalf("SelectValue(city) = %q, %v; want empty initial choice", got, ok)
	}
	sel := w.Find("city").(*widget.Select)
	sel.SetSelected("Bandung")
	got, ok = w.SelectValue("city")
	if !ok || got != "bdg" {
		t.Fatalf("SelectValue(city) = %q, %v; want bdg, true", got, ok)
	}
	if _, ok := w.SelectValue("nama"); ok {
		t.Fatal("SelectValue of an entry reported true")
	}
}

func TestBuildStyle(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><label id="lbl" text="Hi" style="w-full p-2 pt-4"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box, ok := w.Content.(*fyne.Container)
	if !ok {
		t.Fatalf("content is %T, want *fyne.Container", w.Content)
	}
	wrap, ok := box.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("styled child is %T, want padded *fyne.Container", box.Objects[0])
	}
	if _, ok := w.Find("lbl").(*widget.Label); !ok {
		t.Fatalf("Find(lbl) is %T, want *widget.Label", w.Find("lbl"))
	}
	box.Layout.Layout(box.Objects, fyne.NewSize(200, 100))
	if wrap.Size().Width != 200 {
		t.Fatalf("wrapper width = %v, want 200", wrap.Size().Width)
	}
	lbl := wrap.Objects[0]
	if lbl.Position() != fyne.NewPos(8, 16) {
		t.Fatalf("label pos = %v, want (8,16)", lbl.Position())
	}
	if lbl.Size().Width != 184 {
		t.Fatalf("label width = %v, want 184", lbl.Size().Width)
	}
}

func TestBuildMargin(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><label id="a" text="A" style="m-2"/><label id="c" text="C" style="p-1 m-1"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	marginWrap, ok := box.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("margin child is %T, want *fyne.Container", box.Objects[0])
	}
	if _, ok := w.Find("a").(*widget.Label); !ok {
		t.Fatalf("Find(a) is %T, want *widget.Label", w.Find("a"))
	}
	box.Layout.Layout(box.Objects, fyne.NewSize(200, 100))
	if marginWrap.Position() != fyne.NewPos(0, 0) {
		t.Fatalf("margin wrapper pos = %v, want (0,0)", marginWrap.Position())
	}
	if lbl := marginWrap.Objects[0]; lbl.Position() != fyne.NewPos(8, 8) {
		t.Fatalf("label pos = %v, want (8,8)", lbl.Position())
	}
	// Padding inside, margin outside: two nested wrappers.
	both := box.Objects[1].(*fyne.Container)
	innerPad, ok := both.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("margin wrapper child is %T, want padding *fyne.Container", both.Objects[0])
	}
	if lbl := innerPad.Objects[0]; lbl.Position() != fyne.NewPos(4, 4) {
		t.Fatalf("padded label pos = %v, want (4,4)", lbl.Position())
	}
	if _, ok := w.Find("c").(*widget.Label); !ok {
		t.Fatalf("Find(c) is %T, want *widget.Label", w.Find("c"))
	}
}

func TestBuildHBoxFullWidth(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<hbox id="b"><label id="a" text="A"/><label id="c" text="C" style="w-full"/></hbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(300, 60))
	plain := box.Objects[0].Size().Width
	full := box.Objects[1].Size().Width
	if full <= plain || full < 200 {
		t.Fatalf("w-full in hbox = %v, plain = %v; want full to take leftover width", full, plain)
	}
}

// In a grid the final cell height is the row height: h-N only raises
// the row minimum, it never disables the stretch.
func TestBuildGridStretchWins(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox id="g" column="grid" min="1" max="1">` +
		`<rect id="r" color="#2e2e2e" style="h-2"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(200, 400))
	r := box.Objects[0]
	if got := r.Size().Height; got != 400 { // single row, no padding: row height wins
		t.Fatalf("grid cell height = %v, want 400 (row height wins over h-2)", got)
	}
}

func TestBuildGrid(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="g" column="grid" min="1" max="5">` +
		`<label id="a" text="A"/><label id="b" text="B"/><label id="c" text="C"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box, ok := w.Content.(*fyne.Container)
	if !ok {
		t.Fatalf("content is %T, want *fyne.Container", w.Content)
	}
	if len(box.Objects) != 3 {
		t.Fatalf("content children = %d, want 3", len(box.Objects))
	}
	if w.Find("b") == nil {
		t.Fatal("Find(b) did not find the grid child")
	}
	box.Layout.Layout(box.Objects, fyne.NewSize(400, 100))
	y0 := box.Objects[0].Position().Y
	for _, o := range box.Objects {
		if o.Position().Y != y0 {
			t.Fatalf("child Y = %v, want single row at %v", o.Position().Y, y0)
		}
	}
}

func closeEnough(a, b float32) bool {
	d := a - b
	return d < 0.01 && d > -0.01
}

func TestBuildAlignVariants(t *testing.T) {
	for _, align := range []string{"start", "center", "end"} {
		doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
			`<vbox id="b" align="` + align + `" style="w-full">` +
			`<label id="wide" text="A much longer label text"/>` +
			`<label id="narrow" text="A"/></vbox></window></goexml>`
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode %s returned error: %v", align, err)
		}
		w, err := Build(root, map[string]func(){}, nil, nil)
		if err != nil {
			t.Fatalf("Build %s returned error: %v", align, err)
		}
		box := w.Content.(*fyne.Container)
		box.Layout.Layout(box.Objects, fyne.NewSize(400, 100))
		narrow := box.Objects[1]
		nx, nw := narrow.Position().X, narrow.Size().Width
		switch align {
		case "start":
			if nx != 0 {
				t.Fatalf("vbox start: narrow X = %v, want 0", nx)
			}
		case "center":
			if nx <= 0 || !closeEnough(nx*2+nw, 400) {
				t.Fatalf("vbox center: narrow not centered: X=%v W=%v", nx, nw)
			}
		case "end":
			if nx <= 0 || !closeEnough(nx+nw, 400) {
				t.Fatalf("vbox end: narrow not flush right: X=%v W=%v", nx, nw)
			}
		}
	}
	for _, align := range []string{"start", "center", "end"} {
		doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
			`<hbox id="b" align="` + align + `" style="w-full">` +
			`<entry id="tall" text="x" style="h-full"/><label id="short" text="Hi"/></hbox></window></goexml>`
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode %s returned error: %v", align, err)
		}
		w, err := Build(root, map[string]func(){}, nil, nil)
		if err != nil {
			t.Fatalf("Build %s returned error: %v", align, err)
		}
		box := w.Content.(*fyne.Container)
		box.Layout.Layout(box.Objects, fyne.NewSize(400, 200))
		tall, short := box.Objects[0], box.Objects[1]
		if tall.Size().Height != 200 {
			t.Fatalf("hbox %s: tall entry H = %v, want full 200", align, tall.Size().Height)
		}
		sy, sh := short.Position().Y, short.Size().Height
		switch align {
		case "start":
			if sy != 0 {
				t.Fatalf("hbox start: short Y = %v, want 0", sy)
			}
		case "center":
			if sy <= 0 || !closeEnough(sy*2+sh, 200) {
				t.Fatalf("hbox center: short not centered vertically: Y=%v H=%v", sy, sh)
			}
		case "end":
			if sy <= 0 || !closeEnough(sy+sh, 200) {
				t.Fatalf("hbox end: short not flush bottom: Y=%v H=%v", sy, sh)
			}
		}
	}
}

func TestBuildJustify(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<hbox id="b" justify="space-between"><label id="a" text="A"/><label id="c" text="C"/></hbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(300, 60))
	first, last := box.Objects[0], box.Objects[1]
	if first.Position().X != 0 {
		t.Fatalf("first child X = %v, want 0", first.Position().X)
	}
	if end := last.Position().X + last.Size().Width; end != 300 {
		t.Fatalf("last child ends at %v, want 300", end)
	}
}

func TestBuildRect(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><rect id="band" color="#2e2e2e" style="w-full h-2"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	r, ok := w.Find("band").(*canvas.Rectangle)
	if !ok {
		t.Fatalf("Find(band) is %T, want *canvas.Rectangle", w.Find("band"))
	}
	if got, want := r.FillColor, (color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 255}); got != want {
		t.Fatalf("FillColor = %#v, want %#v", got, want)
	}
}

func TestBuildTextElement(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
		text string
		wrap fyne.TextWrap
	}{
		{"body defaults to wrap",
			`<goexml version="0.1.0"><window title="x" size="10x10"><vbox id="b"><text id="t">Hello world</text></vbox></window></goexml>`,
			"Hello world", fyne.TextWrapWord},
		{"v attr",
			`<goexml version="0.1.0"><window title="x" size="10x10"><vbox id="b"><text id="t" v="Hello attr" /></vbox></window></goexml>`,
			"Hello attr", fyne.TextWrapWord},
		{"break",
			`<goexml version="0.1.0"><window title="x" size="10x10"><vbox id="b"><text id="t" wrap="break">Hi</text></vbox></window></goexml>`,
			"Hi", fyne.TextWrapBreak},
		{"off",
			`<goexml version="0.1.0"><window title="x" size="10x10"><vbox id="b"><text id="t" wrap="off">Hi</text></vbox></window></goexml>`,
			"Hi", fyne.TextWrapOff},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Decode([]byte(tc.doc))
			if err != nil {
				t.Fatalf("Decode returned error: %v", err)
			}
			w, err := Build(root, map[string]func(){}, nil, nil)
			if err != nil {
				t.Fatalf("Build returned error: %v", err)
			}
			lbl, ok := w.Find("t").(*widget.Label)
			if !ok {
				t.Fatalf("Find(t) is %T, want *widget.Label", w.Find("t"))
			}
			if lbl.Text != tc.text {
				t.Fatalf("text = %q, want %q", lbl.Text, tc.text)
			}
			if lbl.Wrapping != tc.wrap {
				t.Fatalf("wrapping = %v, want %v", lbl.Wrapping, tc.wrap)
			}
		})
	}
}

func TestBuildWrapGrows(t *testing.T) {
	words := strings.Repeat("lorem ipsum dolor sit amet consectetur adipiscing elit ", 60)
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b" style="w-full"><text id="p">` + words + `</text></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(784, 10000))
	h := box.Objects[0].Size().Height
	single := widget.NewLabel("x").MinSize().Height
	if h <= 10*single {
		t.Fatalf("wrapped height = %v, want more than 10 lines (single=%v)", h, single)
	}
}

func TestBuildTenThousandWords(t *testing.T) {
	words := strings.Repeat("lorem ipsum dolor sit amet ", 2000)
	if got := len(strings.Fields(words)); got != 10000 {
		t.Fatalf("fixture has %d words, want 10000", got)
	}
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b" style="w-full"><text id="big">` + words + `</text></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	box.Layout.Layout(box.Objects, fyne.NewSize(784, 100000))
	if h := box.Objects[0].Size().Height; h <= 2000 {
		t.Fatalf("10000-word block height = %v, want > 2000", h)
	}
}

func TestColumnWidths(t *testing.T) {
	ws, rowH := tableMetrics([]string{"Name", "Role"}, [][]string{{"Ann", "Designer"}, {"Bo", "Engineer"}})
	if len(ws) != 2 {
		t.Fatalf("got %d widths, want 2", len(ws))
	}
	if ws[1] <= ws[0] {
		t.Fatalf("long column %v should exceed short column %v", ws[1], ws[0])
	}
	if ws[0] <= 0 || ws[1] <= 0 {
		t.Fatalf("widths must be positive, got %v", ws)
	}
	if rowH <= 0 {
		t.Fatalf("row height must be positive, got %v", rowH)
	}
	plain, _ := tableMetrics([]string{"A"}, [][]string{{"B"}})
	if plain[0] <= 16 {
		t.Fatalf("single column width %v should include padding", plain[0])
	}
}

func TestBuildExampleFiles(t *testing.T) {
	files, err := filepath.Glob("examples/*.xml")
	if err != nil {
		t.Fatalf("listing examples: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no example files found")
	}
	handlers := map[string]func(){
		"OnCancel": func() {}, "OnSave": func() {},
		"OnPick": func() {}, "OnRow": func() {},
		"OnDemo": func() {}, "OnGreet": func() {},
	}
	// fillContainers builds a contents map with a label for every
	// container id in the tree.
	fillContainers := func(root *Node) map[string]fyne.CanvasObject {
		contents := map[string]fyne.CanvasObject{}
		var walk func(n *Node)
		walk = func(n *Node) {
			if n.Type == "container" {
				contents[n.Attr("id")] = widget.NewLabel("example body")
			}
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(root)
		return contents
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading example %s: %v", f, err)
		}
		// Comments may precede the root, so route by trying the
		// document envelope first and falling back to the fragment.
		root, docErr := Decode(data)
		if docErr == nil {
			// showcase.xml carries an <image>: feed it the demo's
			// resource set so every example builds like in production.
			res := map[string]fyne.Resource(nil)
			if strings.Contains(f, "showcase") {
				res = map[string]fyne.Resource{
					"assets/demo.png": fyne.NewStaticResource("demo.png", pngBytes(2, 1, color.NRGBA{A: 255})),
				}
			}
			if _, err := Build(root, handlers, fillContainers(root), res); err != nil {
				t.Fatalf("Build %s returned error: %v", f, err)
			}
			continue
		}
		item, itemErr := DecodeItem(data)
		if itemErr != nil {
			t.Fatalf("Decode %s returned error: %v", f, docErr)
		}
		// A fragment needs a host window to build into.
		host, err := Decode([]byte(`<goexml version="0.1.0"><window title="host" size="1x1">` +
			`<label id="hostlbl" text="x"/></window></goexml>`))
		if err != nil {
			t.Fatalf("host decode: %v", err)
		}
		w, err := Build(host, handlers, nil, nil)
		if err != nil {
			t.Fatalf("host Build: %v", err)
		}
		if _, err := w.BuildItem(item, fillContainers(item), nil); err != nil {
			t.Fatalf("BuildItem %s returned error: %v", f, err)
		}
	}
}
