package goexml

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func containerDoc(id string) string {
	return `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<vbox id="b">` +
		`<container id="` + id + `"><vbox id="wrap"><label id="h" text="Head"/><slot style="h-2"/><label id="ft" text="Foot"/></vbox></container>` +
		`</vbox></window></goexml>`
}

// Build requires content for every container and injects it into the
// slot; Find returns the *Container handle.
func TestBuildContainerSlot(t *testing.T) {
	root, err := Decode([]byte(containerDoc("page")))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	body := widget.NewLabel("body")
	w, err := Build(root, map[string]func(){}, map[string]fyne.CanvasObject{"page": body})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	c, ok := w.Find("page").(*Container)
	if !ok {
		t.Fatalf("Find(%q) returned %T, want *Container", "page", w.Find("page"))
	}
	if got := c.slot.Objects; len(got) != 1 || got[0] != body {
		t.Fatalf("slot content = %v, want the injected label", got)
	}
	// The slot measures to its content before the parent lays it out.
	if got := c.slot.MinSize().Height; got != body.MinSize().Height {
		t.Fatalf("slot min height = %v, want %v", got, body.MinSize().Height)
	}
	// SetContent replaces; the design stays.
	other := widget.NewLabel("other")
	c.SetContent(other)
	if c.slot.Objects[0] != other {
		t.Fatal("SetContent did not replace the slot content")
	}
}

// An unfilled slot or an unknown contents key is an error.
func TestBuildContainerRequiresContent(t *testing.T) {
	root, err := Decode([]byte(containerDoc("page")))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if _, err := Build(root, map[string]func(){}, nil); err == nil ||
		!strings.Contains(err.Error(), `slot must be filled`) {
		t.Fatalf("unfilled slot error = %v", err)
	}
	if _, err := Build(root, map[string]func(){}, map[string]fyne.CanvasObject{"paeg": widget.NewLabel("x")}); err == nil ||
		!strings.Contains(err.Error(), `no container with id "paeg"`) {
		t.Fatalf("unknown key error = %v", err)
	}
}

// BuildItem builds a fragment into the window's id space.
func TestBuildItem(t *testing.T) {
	root, err := Decode([]byte(containerDoc("page")))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){}, map[string]fyne.CanvasObject{"page": widget.NewLabel("x")})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item, err := DecodeItem([]byte(`<goexml-item version="0.1.0">` +
		`<vbox id="page2"><label id="lbl" text="from item"/></vbox></goexml-item>`))
	if item == nil {
		t.Fatal("DecodeItem failed")
	}
	obj, err := w.BuildItem(item, nil)
	if err != nil {
		t.Fatalf("BuildItem: %v", err)
	}
	if obj == nil {
		t.Fatal("BuildItem returned nil object")
	}
	if _, err := w.Text("lbl"); err != nil {
		t.Fatalf("fragment id not registered: %v", err)
	}

	// Colliding with a document id is rejected.
	bad, err := DecodeItem([]byte(`<goexml-item version="0.1.0"><label id="b" text="x"/></goexml-item>`))
	if _, err := w.BuildItem(bad, nil); err == nil ||
		!strings.Contains(err.Error(), `id "b" is already used by the document`) {
		t.Fatalf("collision error = %v", err)
	}

	// A container in the fragment must be filled; unknown keys rejected.
	frag, err := DecodeItem([]byte(`<goexml-item version="0.1.0"><container id="c1"><slot/></container></goexml-item>`))
	if _, err := w.BuildItem(frag, nil); err == nil ||
		!strings.Contains(err.Error(), "slot must be filled") {
		t.Fatalf("unfilled fragment slot error = %v", err)
	}
	if _, err := w.BuildItem(frag, map[string]fyne.CanvasObject{"nope": widget.NewLabel("x")}); err == nil ||
		!strings.Contains(err.Error(), `no container with id "nope" in the fragment`) {
		t.Fatalf("unknown fragment key error = %v", err)
	}
	if _, err := w.BuildItem(frag, map[string]fyne.CanvasObject{"c1": widget.NewLabel("x")}); err != nil {
		t.Fatalf("fragment container rejected: %v", err)
	}
	if c, ok := w.Find("c1").(*Container); !ok || c.slot.Objects[0] == nil {
		t.Fatal("fragment container not reachable through Find")
	}
}

// Setters dispatch by widget kind with node-named errors.
func TestWindowSetters(t *testing.T) {
	root, err := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="b"><label id="g" text="hi"/><entry id="e" text=""/><select id="s" onChange="P">` +
		`<option value="1" text="One"/><option value="2" text="Two"/></select>` +
		`<rect id="r" color="#111111"/></vbox></window></goexml>`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){"P": func() {}}, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := w.SetText("g", "changed"); err != nil {
		t.Fatalf("SetText: %v", err)
	}
	if got, _ := w.Text("g"); got != "changed" {
		t.Fatalf("Text = %q, want changed", got)
	}
	if err := w.SetText("e", "typed"); err != nil {
		t.Fatalf("SetText entry: %v", err)
	}
	if got, _ := w.Text("e"); got != "typed" {
		t.Fatalf("entry Text = %q, want typed", got)
	}
	if err := w.SetValue("s", "2"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if got, _ := w.SelectValue("s"); got != "2" {
		t.Fatalf("SelectValue = %q, want 2", got)
	}
	// Kind errors are node-named, never panics.
	if err := w.SetText("r", "x"); err == nil || !strings.Contains(err.Error(), `cannot set text on <rect>`) {
		t.Fatalf("kind error = %v", err)
	}
	if err := w.SetText("nope", "x"); err == nil || !strings.Contains(err.Error(), `no widget with id "nope"`) {
		t.Fatalf("unknown id error = %v", err)
	}
	if err := w.SetValue("s", "9"); err == nil || !strings.Contains(err.Error(), `no option with value "9"`) {
		t.Fatalf("bad value error = %v", err)
	}
}

// {{placeholder}} templates render from the raw XML text.
func TestWindowTemplates(t *testing.T) {
	root, err := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="b"><label id="g" text="Halo {{name}}, {{n}}"/>` +
		`<select id="s" onChange="P"><option value="1" text="Item {{i}}"/></select></vbox></window></goexml>`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){"P": func() {}}, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Before rendering, the raw template shows.
	if got, _ := w.Text("g"); got != "Halo {{name}}, {{n}}" {
		t.Fatalf("raw template = %q", got)
	}
	if err := w.SetTextFormat("g", map[string]string{"name": "Budi", "n": "3"}); err != nil {
		t.Fatalf("SetTextFormat: %v", err)
	}
	if got, _ := w.Text("g"); got != "Halo Budi, 3" {
		t.Fatalf("rendered = %q", got)
	}
	if err := w.SetTextFormat("g", map[string]string{"name": "Budi"}); err == nil ||
		!strings.Contains(err.Error(), `missing placeholder "n"`) {
		t.Fatalf("missing token error = %v", err)
	}
	// A plain SetText removes the template registration.
	if err := w.SetText("g", "plain"); err != nil {
		t.Fatalf("SetText: %v", err)
	}
	if err := w.SetTextFormat("g", nil); err == nil ||
		!strings.Contains(err.Error(), `no {{placeholder}} template for id "g"`) {
		t.Fatalf("removed template error = %v", err)
	}
	// Select options render and keep the value mapping.
	s := w.Find("s").(*widget.Select)
	if err := w.SetValueFormat("s", map[string]string{"i": "7"}); err != nil {
		t.Fatalf("SetValueFormat: %v", err)
	}
	if len(s.Options) != 1 || s.Options[0] != "Item 7" {
		t.Fatalf("options = %v", s.Options)
	}
	s.Selected = "Item 7"
	if got, _ := w.SelectValue("s"); got != "1" {
		t.Fatalf("SelectValue after render = %q, want 1", got)
	}
}

// The slot is a style-only leaf: its h-N decides the area the parent
// gives it, and the injected content stretches to fill that area.
func TestSlotHeightStyle(t *testing.T) {
	root, err := Decode([]byte(containerDoc("page")))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	body := widget.NewLabel("body")
	w, err := Build(root, map[string]func(){}, map[string]fyne.CanvasObject{"page": body})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Lay out once: heights only settle after the parent lays out.
	w.Content.Resize(w.Size)
	c := w.Find("page").(*Container)
	// h-2 = 2*4 px, exactly like any other styled leaf.
	if got := c.slot.Size().Height; got != 8 {
		t.Fatalf("slot height = %v, want 8 (h-2)", got)
	}
	if got := c.slot.Objects[0].Size().Height; got != 8 {
		t.Fatalf("slot content height = %v, want 8 (content fills the slot)", got)
	}
}

// Sibling containers are independent injection units in one window.
func TestSiblingContainers(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="b"><container id="head"><vbox id="wh"><label id="h" text="H"/><slot/></vbox></container>` +
		`<container id="body"><vbox id="wb"><label id="f" text="F"/><slot/></vbox></container></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	head, body := widget.NewLabel("head"), widget.NewLabel("body")
	w, err := Build(root, map[string]func(){}, map[string]fyne.CanvasObject{
		"head": head, "body": body,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := w.Find("head").(*Container).slot.Objects[0]; got != head {
		t.Fatal("head slot does not hold its content")
	}
	if got := w.Find("body").(*Container).slot.Objects[0]; got != body {
		t.Fatal("body slot does not hold its content")
	}
}

// Rebuilding the same fragment replaces its ids and slot content: the
// last built page wins, which is the rebuild-per-switch lifecycle.
func TestFragmentRebuildOverwrite(t *testing.T) {
	root, err := Decode([]byte(containerDoc("page")))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){}, map[string]fyne.CanvasObject{"page": widget.NewLabel("x")})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	itemSrc := `<goexml-item version="0.1.0"><container id="c1"><vbox id="f">` +
		`<label id="lbl" text="v1"/><slot/></vbox></container></goexml-item>`
	item1, err := DecodeItem([]byte(itemSrc))
	if err != nil {
		t.Fatalf("DecodeItem: %v", err)
	}
	if _, err := w.BuildItem(item1, map[string]fyne.CanvasObject{"c1": widget.NewLabel("p1")}); err != nil {
		t.Fatalf("BuildItem 1: %v", err)
	}
	if got, _ := w.Text("lbl"); got != "v1" {
		t.Fatalf("first build text = %q, want v1", got)
	}
	// Same fragment source, built again: ids are re-registered, not
	// rejected, and the new content replaces the old.
	item2, err := DecodeItem([]byte(itemSrc))
	if err != nil {
		t.Fatalf("DecodeItem 2: %v", err)
	}
	second := widget.NewLabel("p2")
	if _, err := w.BuildItem(item2, map[string]fyne.CanvasObject{"c1": second}); err != nil {
		t.Fatalf("BuildItem 2: %v", err)
	}
	if got, _ := w.Text("lbl"); got != "v1" {
		t.Fatalf("rebuilt fragment text = %q, want v1 (fresh widget)", got)
	}
	if got := w.Find("c1").(*Container).slot.Objects[0]; got != second {
		t.Fatal("rebuilt fragment slot does not hold the new content")
	}
}
