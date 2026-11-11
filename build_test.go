package goexml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
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
	w, err := Build(root, tap)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	return w
}

func TestBuildForm(t *testing.T) {
	saved := false
	w := buildValid(t, map[string]func(){
		"OnBatal":  func() {},
		"OnSimpan": func() { saved = true },
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
		t.Fatal("tapping save did not call OnSimpan")
	}
}

func TestBuildUnknownHandler(t *testing.T) {
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	_, err = Build(root, map[string]func(){})
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
	w, err := Build(root, map[string]func(){})
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
	w, err := Build(root, map[string]func(){})
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
		`<option value="a" text="A"/><option value="b" text="B" selected="true"/></select>` +
		`<table id="t"><row><cell text="Name"/><cell text="Age"/></row>` +
		`<row><cell text="Ann"/><cell text="30"/></row>` +
		`<row><cell text="Bo"/><cell text="25"/></row></table>` +
		`</vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	picked := ""
	_, err = Build(root, map[string]func(){})
	if err == nil {
		t.Fatal("expected no-handler error for missing OnPick, got nil")
	}
	pick := func() { picked = "yes" }
	w, err := Build(root, map[string]func(){"OnPick": pick})
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
	tab, ok := box.Objects[1].(*widget.Table)
	if !ok {
		t.Fatalf("second child is %T, want *widget.Table", box.Objects[1])
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
	if _, err := Build(root, map[string]func(){}); err == nil {
		t.Fatal("expected no-handler error for missing OnRow, got nil")
	}
	chosen := false
	w, err := Build(root, map[string]func(){"OnRow": func() { chosen = true }})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	tab, ok := w.Content.(*widget.Table)
	if !ok {
		t.Fatalf("content is %T, want *widget.Table", w.Content)
	}
	if tab.OnSelected == nil {
		t.Fatal("table has no OnSelected handler")
	}
	tab.OnSelected(widget.TableCellID{Row: 0, Col: 1})
	if !chosen {
		t.Fatal("selecting a cell did not call OnRow")
	}
}

func TestWindowFindSelectValue(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><select id="kota" onChange="OnPick">` +
		`<option value="jkt" text="Jakarta"/><option value="bdg" text="Bandung"/></select>` +
		`<entry id="nama"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){"OnPick": func() {}})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if w.Find("hilang") != nil {
		t.Fatal("Find of unknown id did not return nil")
	}
	if _, ok := w.Find("b").(*fyne.Container); !ok {
		t.Fatalf("Find(b) is %T, want *fyne.Container", w.Find("b"))
	}
	got, ok := w.SelectValue("kota")
	if !ok || got != "" {
		t.Fatalf("SelectValue(kota) = %q, %v; want empty initial choice", got, ok)
	}
	sel := w.Find("kota").(*widget.Select)
	sel.SetSelected("Bandung")
	got, ok = w.SelectValue("kota")
	if !ok || got != "Bandung" {
		t.Fatalf("SelectValue(kota) = %q, %v; want Bandung, true", got, ok)
	}
	if _, ok := w.SelectValue("nama"); ok {
		t.Fatal("SelectValue of an entry reported true")
	}
}

func TestBuildStyle(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><label id="lbl" text="Hi" style="full-width p-2 pt-4"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){})
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
	w, err := Build(root, map[string]func(){})
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
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading example %s: %v", f, err)
		}
		root, err := Decode(data)
		if err != nil {
			t.Fatalf("Decode %s returned error: %v", f, err)
		}
		if _, err := Build(root, handlers); err != nil {
			t.Fatalf("Build %s returned error: %v", f, err)
		}
	}
}
