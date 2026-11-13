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

// commitOne drafts frag into l, commits it, and fails the test on any
// error — for fragments with no open slots.
func commitOne(t *testing.T, l *List, frag *Node) *Item {
	t.Helper()
	it, err := l.Draft(frag)
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if err := l.Commit(it); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return it
}

// TestListDraftCommit covers the draft/commit contract: Draft builds
// the fragment off-screen with a private id space, the app fills it,
// Commit stacks a complete row — and every broken input is an
// explicit error.
func TestListDraftCommit(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="feed" style="bg-[#112233] rounded-md" scrollable /></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	res := map[string]fyne.Resource{
		"pic": fyne.NewStaticResource("pic.png", listPng()),
	}
	w, err := Build(root, map[string]func(){"OnRow": func() {}}, nil, res)
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

	// Deterministic draft errors before anything valid.
	badDoc, _ := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x"/></window></goexml>`))
	for _, tc := range []struct {
		name string
		node *Node
		want string
	}{
		{"nil node", nil, "goexml-item"},
		{"wrong node", badDoc, "goexml-item"},
	} {
		if _, err := feed.Draft(tc.node); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}

	// Draft the same fragment twice: private id spaces, distinct
	// objects, per-item data. Nothing reaches the screen yet.
	first, err := feed.Draft(frag)
	if err != nil {
		t.Fatalf("Draft 1: %v", err)
	}
	second, err := feed.Draft(frag)
	if err != nil {
		t.Fatalf("Draft 2: %v", err)
	}
	if feed.Count() != 0 {
		t.Fatalf("drafts reached the screen early: Count = %d", feed.Count())
	}
	// Ids are private: the window must not see fragment ids...
	if w.Find("rowTitle") != nil {
		t.Fatal("item ids leaked into the window scope")
	}
	// ...while each draft's widgets are distinct and fillable.
	t1, _ := first.Find("rowTitle").(*widget.Label)
	t2, _ := second.Find("rowTitle").(*widget.Label)
	if t1 == nil || t2 == nil || t1 == t2 {
		t.Fatal("drafts share their widgets or miss them")
	}
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
	// The src key resolves from the window's shared resource set.
	if err := first.SetImage("rowPic", "pic"); err != nil {
		t.Fatalf("SetImage: %v", err)
	}

	// Commit refuses an unfilled slot: a committed row is complete.
	if err := feed.Commit(first); err == nil || !strings.Contains(err.Error(), `slot "rowSlot" is not filled`) {
		t.Fatalf("error %q does not name the empty slot", err)
	}
	first.Find("rowSlot").(*Container).SetContent(widget.NewLabel("body 1"))
	if err := feed.Commit(first); err != nil {
		t.Fatalf("Commit 1: %v", err)
	}
	// The second draft still needs its own slot filled.
	second.Find("rowSlot").(*Container).SetContent(widget.NewLabel("body 2"))
	if err := feed.Commit(second); err != nil {
		t.Fatalf("Commit 2: %v", err)
	}
	if feed.Count() != 2 {
		t.Fatalf("Count = %d, want 2", feed.Count())
	}
	// Committing again is refused.
	if err := feed.Commit(second); err == nil || !strings.Contains(err.Error(), "already committed") {
		t.Fatalf("error %q does not refuse the double commit", err)
	}
	// A foreign draft is refused here.
	otherRoot, _ := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="100x100"><vbox-expand id="other"/></window></goexml>`))
	otherWin, _ := Build(otherRoot, map[string]func(){"OnRow": func() {}}, nil, res)
	other := otherWin.Find("other").(*List)
	stray, err := other.Draft(frag)
	if err != nil {
		t.Fatalf("Draft on other: %v", err)
	}
	if err := feed.Commit(stray); err == nil || !strings.Contains(err.Error(), "another list") {
		t.Fatalf("error %q does not refuse a foreign draft", err)
	}

	// Slot images pending at commit time are fine and stay auditable:
	// they keep their size, unlike an empty container slot.
	slotFrag, _ := DecodeItem([]byte(`<goexml-item version="0.1.0"><image id="slot" style="h-4"/></goexml-item>`))
	withSlot, err := feed.Draft(slotFrag)
	if err != nil {
		t.Fatalf("Draft slot frag: %v", err)
	}
	if got := withSlot.UnfilledImages(); len(got) != 1 || got[0] != "slot" {
		t.Fatalf("UnfilledImages = %v, want [slot]", got)
	}
	if err := feed.Commit(withSlot); err != nil {
		t.Fatalf("Commit slot frag: %v", err)
	}
	if got := withSlot.UnfilledImages(); len(got) != 1 {
		t.Fatalf("UnfilledImages after commit = %v, want still pending", got)
	}
	if err := withSlot.SetImageRes("slot", fyne.NewStaticResource("s.png", listPng())); err != nil {
		t.Fatalf("SetImageRes: %v", err)
	}
	if withSlot.UnfilledImages() != nil {
		t.Fatalf("UnfilledImages after fill = %v, want none", withSlot.UnfilledImages())
	}

	// Clear retires the committed items and every pending draft.
	pending, _ := feed.Draft(frag)
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
	if got := withSlot.UnfilledImages(); got != nil {
		t.Fatalf("UnfilledImages on a cleared item = %v, want nil", got)
	}
	if err := feed.Commit(pending); err == nil || !strings.Contains(err.Error(), "cleared") {
		t.Fatalf("error %q does not retire the pending draft", err)
	}
	// The list still works after a clear: draft, fill, commit.
	again, err := feed.Draft(frag)
	if err != nil {
		t.Fatalf("Draft after Clear: %v", err)
	}
	again.Find("rowSlot").(*Container).SetContent(widget.NewLabel("fresh"))
	if err := feed.Commit(again); err != nil {
		t.Fatalf("Commit after Clear: %v", err)
	}
	if err := again.SetText("rowTitle", "new"); err != nil {
		t.Fatalf("SetText on fresh item: %v", err)
	}
}

// TestListItemSetters covers the Item wrappers the list tests never
// touched: select setters (SetValue, SelectValue, SetValueFormat) and
// the per-item image payload (SetImageRes), all scoped to one item.
func TestListItemSetters(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="feed"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, map[string]func(){"OnPick": func() {}}, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	feed := w.Find("feed").(*List)
	frag, err := DecodeItem([]byte(`<goexml-item version="0.1.0"><vbox>` +
		`<select id="pick" onChange="OnPick">` +
		`<option value="a" text="A {{x}}"/><option value="b" text="B {{x}}"/>` +
		`</select>` +
		`<image id="thumb" style="h-4"/>` +
		`</vbox></goexml-item>`))
	if err != nil {
		t.Fatalf("DecodeItem: %v", err)
	}
	it, err := feed.Draft(frag)
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	if err := feed.Commit(it); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// SetValue picks by option value; SelectValue reads it back.
	if err := it.SetValue("pick", "b"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if v, ok := it.SelectValue("pick"); !ok || v != "b" {
		t.Fatalf("SelectValue = %q, %v; want b, true", v, ok)
	}
	// SetValueFormat re-renders the option labels and keeps the
	// selection by value.
	if err := it.SetValueFormat("pick", map[string]string{"x": "2"}); err != nil {
		t.Fatalf("SetValueFormat: %v", err)
	}
	sel := it.Find("pick").(*widget.Select)
	if len(sel.Options) != 2 || sel.Options[0] != "A 2" || sel.Options[1] != "B 2" {
		t.Fatalf("options = %v, want [A 2 B 2]", sel.Options)
	}
	if v, _ := it.SelectValue("pick"); v != "b" {
		t.Fatalf("SelectValue after re-render = %q, want b", v)
	}
	// SetImageRes fills the pending slot image directly.
	if got := it.UnfilledImages(); len(got) != 1 || got[0] != "thumb" {
		t.Fatalf("UnfilledImages = %v, want [thumb]", got)
	}
	if err := it.SetImageRes("thumb", fyne.NewStaticResource("t.png", listPng())); err != nil {
		t.Fatalf("SetImageRes: %v", err)
	}
	if it.UnfilledImages() != nil {
		t.Fatalf("UnfilledImages after fill = %v, want none", it.UnfilledImages())
	}

	// After Clear, every setter is an explicit error - including the
	// wrappers this test exists for.
	feed.Clear()
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"SetValue", func() error { return it.SetValue("pick", "a") }},
		{"SetValueFormat", func() error { return it.SetValueFormat("pick", map[string]string{"x": "3"}) }},
		{"SetImageRes", func() error { return it.SetImageRes("thumb", fyne.NewStaticResource("t.png", listPng())) }},
	} {
		if err := tc.call(); err == nil || !strings.Contains(err.Error(), "cleared") {
			t.Fatalf("%s on a cleared item: error %q does not retire the item", tc.name, err)
		}
	}
	if v, ok := it.SelectValue("pick"); v != "" || ok {
		t.Fatalf("SelectValue on a cleared item = %q, %v; want empty, false", v, ok)
	}
}

// TestListSlotNil covers the nil slot: SetContent(nil) empties the
// slot instead of storing a nil object, so Commit refuses it as an
// unfilled slot — never a nil object that panics the renderer.
func TestListSlotNil(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="feed"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	feed := w.Find("feed").(*List)
	frag, _ := DecodeItem([]byte(`<goexml-item version="0.1.0">` +
		`<container id="slot"><slot/></container></goexml-item>`))
	it, err := feed.Draft(frag)
	if err != nil {
		t.Fatalf("Draft: %v", err)
	}
	it.Find("slot").(*Container).SetContent(nil)
	if err := feed.Commit(it); err == nil || !strings.Contains(err.Error(), `slot "slot" is not filled`) {
		t.Fatalf("error %q does not name the nil slot as unfilled", err)
	}
	if feed.Count() != 0 {
		t.Fatal("a nil slot was committed")
	}
	// The slot is empty, not poisoned: filling it for real commits.
	it.Find("slot").(*Container).SetContent(widget.NewLabel("body"))
	if err := feed.Commit(it); err != nil {
		t.Fatalf("Commit after refill: %v", err)
	}
	if feed.Count() != 1 {
		t.Fatalf("Count = %d, want 1", feed.Count())
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
	commitOne(t, feed, frag)
}

// TestListSpacingAlign locks the lifted static-box layout: spacing
// separates committed items, align positions them on the cross axis,
// and every committed item picks up its style hint through the
// listLayout adapter.
func TestListSpacingAlign(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="feed" spacing="8" align="end"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	feed := w.Find("feed").(*List)
	frag, _ := DecodeItem([]byte(`<goexml-item version="0.1.0"><label text="x" style="h-1"/></goexml-item>`))
	for i := 0; i < 3; i++ {
		commitOne(t, feed, frag)
	}
	// MinSize must see the spacing between the fixed-height items:
	// three h-1 (4px) rows plus two 8px gaps.
	if got := feed.box.MinSize().Height; got != 4*3+8*2 {
		t.Fatalf("MinSize height = %v, want 28", got)
	}
	feed.box.Layout.Layout(feed.box.Objects, fyne.NewSize(200, 100))
	for i, o := range feed.box.Objects {
		if want := float32(i) * 12; o.Position().Y != want {
			t.Fatalf("item %d at y=%v, want %v (4px row + 8px gap)", i, o.Position().Y, want)
		}
		// align="end": the right edge hugs the parent width, whatever
		// the text's measured width is.
		if right := o.Position().X + o.Size().Width; right != 200 {
			t.Fatalf("item %d right edge = %v, want 200", i, right)
		}
	}
}

// TestListGrid locks the grid column block on a committed stack:
// items flow into columns and wrap rows, with per-item hints still
// applied.
func TestListGrid(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="200x400">` +
		`<vbox-expand id="grid" column="grid" min="2" max="2" spacing="4"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := w.Find("grid").(*List)
	frag, _ := DecodeItem([]byte(`<goexml-item version="0.1.0"><label text="x" style="h-1"/></goexml-item>`))
	for i := 0; i < 3; i++ {
		commitOne(t, grid, frag)
	}
	// Layout at the exact content size: two 4px rows plus one 4px
	// gap, so the grid's leftover-height sharing stays out of the
	// math.
	grid.box.Layout.Layout(grid.box.Objects, fyne.NewSize(200, 12))
	// Two columns of 98px with a 4px gutter; the third item wraps to
	// the second row.
	cellW := float32(98)
	if got := grid.box.Objects[0].Size().Width; got != cellW {
		t.Fatalf("cell width = %v, want %v", got, cellW)
	}
	if got := grid.box.Objects[1].Position().X; got != cellW+4 {
		t.Fatalf("column 2 x = %v, want %v", got, cellW+4)
	}
	if got := grid.box.Objects[2].Position().Y; got != 8 {
		t.Fatalf("wrapped row y = %v, want 8 (4px row + 4px gap)", got)
	}
	if got := grid.box.Objects[2].Position().X; got != 0 {
		t.Fatalf("wrapped item x = %v, want 0", got)
	}
}
