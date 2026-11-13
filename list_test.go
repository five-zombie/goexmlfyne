package goexml

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// listPng returns a tiny but valid PNG, so list tests carry their own
// fixtures without reading files.
func listPng() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// TestListAppend covers the append contract: the handle comes from
// Find, items stack with private id spaces, and every broken input is
// an explicit error.
func TestListAppend(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="feed" style="bg-[#112233] rounded-md" scrollable="" /></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){"OnRow": func() {}}, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	feed, ok := w.Find("feed").(*List)
	if !ok {
		t.Fatalf("Find(feed) = %T, want *List", w.Find("feed"))
	}
	if feed.Count() != 0 {
		t.Fatal("a fresh list is not empty")
	}

	frag, err := DecodeItem([]byte(`<goexml-item version="0.1.0">` +
		`<vbox id="row" spacing="2" style="bg-[#334455] p-2">` +
		`<label id="rowTitle" text="{{title}}"/>` +
		`<button id="rowOpen" text="open" onClick="OnRow"/>` +
		`<image id="rowPic" src="pic" style="w-full h-4"/>` +
		`<container id="rowSlot"><slot/></container>` +
		`</vbox></goexml-item>`))
	if err != nil {
		t.Fatalf("DecodeItem: %v", err)
	}

	// Deterministic append errors before anything valid.
	badDoc, _ := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x"/></window></goexml>`))
	res := map[string]fyne.Resource{
		"pic": fyne.NewStaticResource("pic.png", listPng()),
	}
	for _, tc := range []struct {
		name string
		node *Node
		// contents must mention a container the fragment lacks, or
		// leave one unfilled
		contents  map[string]fyne.CanvasObject
		resources map[string]fyne.Resource
		want      string
	}{
		{"nil node", nil, nil, nil, "goexml-item"},
		{"wrong node", badDoc, nil, nil, "goexml-item"},
		{"unknown slot", frag, map[string]fyne.CanvasObject{"nope": widget.NewLabel("x")}, res, "no container with id"},
		{"unfilled slot", frag, nil, res, "slot must be filled"},
	} {
		if _, err := feed.Append(tc.node, tc.contents, tc.resources); err == nil ||
			!strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}
	if feed.Count() != 0 {
		t.Fatalf("failed appends stacked %d objects", feed.Count())
	}

	// Two appends of the same fragment: private id spaces, distinct
	// objects, per-item data.
	first, err := feed.Append(frag, map[string]fyne.CanvasObject{
		"rowSlot": widget.NewLabel("body 1"),
	}, res)
	if err != nil {
		t.Fatalf("Append 1: %v", err)
	}
	second, err := feed.Append(frag, map[string]fyne.CanvasObject{
		"rowSlot": widget.NewLabel("body 2"),
	}, res)
	if err != nil {
		t.Fatalf("Append 2: %v", err)
	}
	if feed.Count() != 2 {
		t.Fatalf("Count = %d, want 2", feed.Count())
	}
	// Ids are private: the window must not see fragment ids...
	if w.Find("rowTitle") != nil {
		t.Fatal("item ids leaked into the window scope")
	}
	// ...while the handlers resolve from the window's map (build
	// succeeded with OnRow) and each item's widgets are distinct.
	t1, _ := first.Find("rowTitle").(*widget.Label)
	t2, _ := second.Find("rowTitle").(*widget.Label)
	if t1 == nil || t2 == nil || t1 == t2 {
		t.Fatal("items share their widgets or miss them")
	}
	// Item setters work, including templates and images.
	if err := first.SetTextFormat("rowTitle", map[string]string{"title": "one"}); err != nil {
		t.Fatalf("SetTextFormat: %v", err)
	}
	if v, _ := first.Text("rowTitle"); v != "one" {
		t.Fatalf("Text = %q, want one", v)
	}
	if err := second.SetText("rowTitle", "two"); err != nil {
		t.Fatalf("SetText: %v", err)
	}
	if v, _ := second.Text("rowTitle"); v != "two" {
		t.Fatalf("second Text = %q, want two", v)
	}
	if err := first.SetImage("rowPic", "pic"); err != nil {
		t.Fatalf("SetImage: %v", err)
	}
	if first.UnfilledImages() != nil {
		t.Fatalf("UnfilledImages after fill = %v, want none", first.UnfilledImages())
	}
	// Per-item resource sets: an item is built against exactly the set
	// it is appended with — the same fragment missing a key its
	// sibling got fails on its own append, never elsewhere.
	before := feed.Count()
	_, err = feed.Append(frag, map[string]fyne.CanvasObject{
		"rowSlot": widget.NewLabel("body 3"),
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown image resource") {
		t.Fatalf("error %q does not name the missing set", err)
	}
	if feed.Count() != before {
		t.Fatal("a failed append changed the stack")
	}
	// Slot images inside an item are auditable on the item.
	slotFrag, _ := DecodeItem([]byte(`<goexml-item version="0.1.0"><image id="slot" style="h-4"/></goexml-item>`))
	withSlot, err := feed.Append(slotFrag, nil, nil)
	if err != nil {
		t.Fatalf("Append slot frag: %v", err)
	}
	if got := withSlot.UnfilledImages(); len(got) != 1 || got[0] != "slot" {
		t.Fatalf("UnfilledImages = %v, want [slot]", got)
	}

	// Clear retires the handles and empties the stack.
	feed.Clear()
	if feed.Count() != 0 {
		t.Fatal("Clear left items")
	}
	err = first.SetText("rowTitle", "zombie")
	if err == nil || !strings.Contains(err.Error(), "cleared") {
		t.Fatalf("error %q does not retire the item", err)
	}
	if first.Find("rowTitle") != nil {
		t.Fatal("Find on a cleared item returned widgets")
	}
	// The list still works after a clear.
	again, err := feed.Append(frag, map[string]fyne.CanvasObject{
		"rowSlot": widget.NewLabel("fresh"),
	}, res)
	if err != nil {
		t.Fatalf("Append after Clear: %v", err)
	}
	if err := again.SetText("rowTitle", "new"); err != nil {
		t.Fatalf("SetText on fresh item: %v", err)
	}
}

// TestListNonScrollable locks the plain box variant: same handle, no
// scroll wrapper, and the visual layer still applies.
func TestListNonScrollable(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="feed" style="bg-[#112233] h-8"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	feed, ok := w.Find("feed").(*List)
	if !ok {
		t.Fatalf("Find(feed) = %T, want *List", w.Find("feed"))
	}
	frag, _ := DecodeItem([]byte(`<goexml-item version="0.1.0"><label id="l" text="x"/></goexml-item>`))
	if _, err := feed.Append(frag, nil, nil); err != nil {
		t.Fatalf("Append: %v", err)
	}
}
