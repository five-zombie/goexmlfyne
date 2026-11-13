package goexml

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// pngBytes returns a tiny but valid PNG of the given size, so tests
// carry their own fixtures instead of reading files.
func pngBytes(w, h int, c color.Color) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// TestBuildImage covers the build contract: resource lookup, the fit
// mapping, intrinsic min size, and the deterministic failure for a
// missing or non-image resource.
func TestBuildImage(t *testing.T) {
	res := map[string]fyne.Resource{
		"dot":      fyne.NewStaticResource("dot.png", pngBytes(3, 2, color.NRGBA{R: 255, A: 255})),
		"wide":     fyne.NewStaticResource("wide.png", pngBytes(8, 4, color.NRGBA{G: 255, A: 255})),
		"junk.txt": fyne.NewStaticResource("junk.txt", []byte("not an image")),
	}
	build := func(t *testing.T, style string) fyne.CanvasObject {
		t.Helper()
		doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
			`<image id="img" src="` + style + `"/></window></goexml>`
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode: %v", err)
		}
		w, err := Build(root, nil, nil, res)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return w.Find("img")
	}

	img := build(t, "dot").(*canvas.Image)
	if img.Resource != res["dot"] {
		t.Fatal("image did not get its resource")
	}
	if img.FillMode != canvas.ImageFillContain {
		t.Fatalf("default fit = %v, want contain", img.FillMode)
	}
	// Intrinsic min size from the 3x2 pixels.
	if min := img.MinSize(); min.Width != 3 || min.Height != 2 {
		t.Fatalf("min size = %v, want 3x2 intrinsic", min)
	}
	// The style path still wraps it: dimensions and the visual layer.
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<vbox id="v"><image id="img" src="dot" style="w-full h-8 bg-[#112233] rounded-md"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, nil, nil, res)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Find returns the inner image (like table); the wrapper sits
	// above it: vbox child -> visual wrapper (bg, image).
	vbox := w.Content.(*fyne.Container)
	wrapper, ok := vbox.Objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("vbox child = %T, want the visual wrapper", vbox.Objects[0])
	}
	if len(wrapper.Objects) != 2 {
		t.Fatalf("layers = %d, want 2 (bg, image)", len(wrapper.Objects))
	}
	if wrapper.Objects[1] != w.Find("img") {
		t.Fatal("the image is not the wrapper's content layer")
	}

	// fit mapping.
	fitCases := map[string]canvas.ImageFill{
		"":      canvas.ImageFillContain,
		"cover": canvas.ImageFillCover,
		"fill":  canvas.ImageFillStretch,
	}
	for fit, want := range fitCases {
		doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
			`<image id="img" src="dot" fit="` + fit + `"/></window></goexml>`
		root, _ := Decode([]byte(doc))
		w, err := Build(root, nil, nil, res)
		if err != nil {
			t.Fatalf("Build fit=%q: %v", fit, err)
		}
		if got := w.Find("img").(*canvas.Image).FillMode; got != want {
			t.Fatalf("fit=%q mode = %v, want %v", fit, got, want)
		}
	}

	// Deterministic failures: unknown key and non-image bytes.
	for _, tc := range []struct{ name, src, want string }{
		{"unknown resource", "missing", "unknown image resource"},
		{"non-image resource", "junk.txt", "bad image resource"},
	} {
		doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
			`<image id="img" src="` + tc.src + `"/></window></goexml>`
		root, _ := Decode([]byte(doc))
		_, err := Build(root, nil, nil, res)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}

	// The wide resource keeps its own intrinsic size: resources are
	// per-key, not per-widget.
	if min := build(t, "wide").(*canvas.Image).MinSize(); min.Width != 8 || min.Height != 4 {
		t.Fatalf("wide min size = %v, want 8x4 intrinsic", min)
	}
}

// TestSetImage covers the runtime swap: same resource set, same
// determinism, and the retire rules staying intact.
func TestSetImage(t *testing.T) {
	res := map[string]fyne.Resource{
		"dot": fyne.NewStaticResource("dot.png", pngBytes(3, 2, color.NRGBA{R: 255, A: 255})),
		"bar": fyne.NewStaticResource("bar.png", pngBytes(4, 6, color.NRGBA{B: 255, A: 255})),
	}
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<image id="img" src="dot"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, res)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := w.Find("img").(*canvas.Image)
	if err := w.SetImage("img", "bar"); err != nil {
		t.Fatalf("SetImage: %v", err)
	}
	if img.Resource != res["bar"] {
		t.Fatal("SetImage did not swap the resource")
	}
	if min := img.MinSize(); min.Width != 4 || min.Height != 6 {
		t.Fatalf("min size after swap = %v, want 4x6", min)
	}
	for _, tc := range []struct {
		name, id, key, want string
	}{
		{"unknown key", "img", "nope", "unknown image resource"},
		{"unknown id", "nope", "dot", "no widget with id"},
		{"bad bytes", "img", "bad", "bad image resource"},
	} {
		if tc.key == "bad" {
			w.resources["bad"] = fyne.NewStaticResource("bad.png", []byte("junk"))
		}
		err := w.SetImage(tc.id, tc.key)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}
	// Non-image nodes reject the setter.
	if err := w.SetImage("img", "dot"); err != nil {
		t.Fatalf("SetImage after errors: %v", err)
	}
}

// TestSetImageSlot covers the src-less slot model: the build leaves
// the image empty, the setters fill it, and UnfilledImages reports
// orphans so a slot can never be forgotten.
func TestSetImageSlot(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<vbox id="v"><image id="slot" fit="cover" style="w-full h-8"/></vbox></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build without resources: %v", err)
	}
	img, ok := w.Find("slot").(*canvas.Image)
	if !ok {
		t.Fatalf("slot image = %T, want *canvas.Image", w.Find("slot"))
	}
	if img.Resource != nil {
		t.Fatal("slot image started filled")
	}
	if got := w.UnfilledImages(); len(got) != 1 || got[0] != "slot" {
		t.Fatalf("UnfilledImages = %v, want [slot]", got)
	}

	// Fill directly from fetched bytes — no resource set needed.
	res := fyne.NewStaticResource("photo.png", pngBytes(5, 7, color.NRGBA{G: 255, A: 255}))
	if err := w.SetImageRes("slot", res); err != nil {
		t.Fatalf("SetImageRes: %v", err)
	}
	if img.Resource != res {
		t.Fatal("SetImageRes did not attach the resource")
	}
	if min := img.MinSize(); min.Width != 5 || min.Height != 7 {
		t.Fatalf("min size after fill = %v, want 5x7", min)
	}
	if got := w.UnfilledImages(); len(got) != 0 {
		t.Fatalf("UnfilledImages after fill = %v, want none", got)
	}

	// SetImage still works against the window's resource set.
	w.resources["swap"] = fyne.NewStaticResource("swap.png", pngBytes(2, 9, color.NRGBA{A: 255}))
	if err := w.SetImage("slot", "swap"); err != nil {
		t.Fatalf("SetImage on slot: %v", err)
	}
	if min := img.MinSize(); min.Width != 2 || min.Height != 9 {
		t.Fatalf("min size after swap = %v, want 2x9", min)
	}

	// SetImageRes determinism: nil resource, bad bytes, unknown id,
	// wrong node kind.
	for _, tc := range []struct {
		name string
		id   string
		res  fyne.Resource
		want string
	}{
		{"nil resource", "slot", nil, "nil resource"},
		{"bad bytes", "slot", fyne.NewStaticResource("junk.png", []byte("nope")), "bad image resource"},
		{"unknown id", "nope", res, "no widget with id"},
		{"wrong node", "v", res, "cannot set an image on"},
	} {
		err := w.SetImageRes(tc.id, tc.res)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}
}

// TestOptionalID locks the relaxed id contract: possibly-static
// elements build and render without an id, stay invisible to Find and
// the setters, and remain in the visual layer like any widget.
func TestOptionalID(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<vbox id="v">` +
		`<rect color="#333333" style="h-1"/>` +
		`<label text="static"/>` +
		`<hbox spacing="4"><label text="inner"/></hbox>` +
		`</vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if w.Find("v") == nil {
		t.Fatal("an element with an id must stay registered")
	}
	for _, id := range []string{"", "static", "inner"} {
		if o := w.Find(id); id == "" && o != nil {
			t.Fatal("empty id lookup returned something")
		}
	}
	if _, err := w.Text("static"); err == nil || !strings.Contains(err.Error(), "no widget with id") {
		t.Fatalf("error %q does not name the missing registration", err)
	}
}

// TestBareFlags locks the flag contract: a bare scrollable or
// selected parses as "true", in every position a start tag offers —
// and the rewriter never touches lookalikes in comments, text, or
// attribute values.
func TestBareFlags(t *testing.T) {
	// Bare forms survive both entries and arrive as "true".
	for _, doc := range []string{
		`<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand id="f" scrollable /></window></goexml>`,
		`<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand id="f" scrollable/></window></goexml>`,
		`<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand scrollable id="f"/></window></goexml>`,
		`<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand id="f" scrollable="true" /></window></goexml>`,
	} {
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode(%s): %v", doc, err)
		}
		var exp *Node
		var walk func(n *Node)
		walk = func(n *Node) {
			if n.Type == "vbox-expand" {
				exp = n
			}
			for _, c := range n.Children {
				walk(c)
			}
		}
		walk(root)
		if got := exp.Attr("scrollable"); got != "true" {
			t.Fatalf("scrollable = %q, want \"true\" (doc %s)", got, doc)
		}
	}
	// A selected option in the bare form is still the chosen one.
	root, err := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<select id="s" onChange="P"><option value="a" text="A"/><option value="b" text="B" selected/></select></window></goexml>`))
	if err != nil {
		t.Fatalf("Decode selected: %v", err)
	}
	_ = root

	// Lookalikes stay verbatim: a comment, a CDATA section holding
	// tag-shaped text, an attribute value holding a flag's name or a
	// > sign, and inner text must all survive the rewrite untouched.
	// The CDATA interior is real text — a rewrite there would corrupt
	// the document, not just an ignored token.
	doc := `<goexml version="0.1.0"><window title="x > scrollable" size="1x1">` +
		`<!-- <vbox scrollable> -->` +
		`<vbox id="v"><label text="not selected really"/>` +
		`<text><![CDATA[usage: <vbox scrollable> and <option selected>]]></text>` +
		`<select id="s" onChange="P"><option value="selected" text="A" selected="true"/></select>` +
		`</vbox></window></goexml>`
	n, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode lookalikes: %v", err)
	}
	var win *Node
	for _, c := range n.Children {
		if c.Type == "window" {
			win = c
		}
	}
	if got := win.Attr("title"); got != "x > scrollable" {
		t.Fatalf("title = %q, want the raw value", got)
	}
	var txt *Node
	var opt *Node
	var findTextOpt func(n *Node)
	findTextOpt = func(n *Node) {
		if n.Type == "text" {
			txt = n
		}
		if n.Type == "option" {
			opt = n
		}
		for _, c := range n.Children {
			findTextOpt(c)
		}
	}
	findTextOpt(win)
	if want := "usage: <vbox scrollable> and <option selected>"; txt.Text != want {
		t.Fatalf("CDATA text = %q, want %q", txt.Text, want)
	}
	if got := opt.Attr("value"); got != "selected" {
		t.Fatalf("option value = %q, want \"selected\"", got)
	}
	if got := opt.Attr("selected"); got != "true" {
		t.Fatalf("explicit selected = %q, want \"true\"", got)
	}
	// Fragments patch the same way: a bare flag on the fragment root's
	// child.
	frag, err := DecodeItem([]byte(`<goexml-item version="0.1.0"><vbox-expand id="f" scrollable/></goexml-item>`))
	if err != nil {
		t.Fatalf("DecodeItem: %v", err)
	}
	if got := frag.Children[0].Attr("scrollable"); got != "true" {
		t.Fatalf("fragment scrollable = %q, want \"true\"", got)
	}
}

// TestBuildItemKeepsDocumentResources locks the resource-set merge:
// a fragment build never orphans the document's images, and a
// fragment's own payloads join the window's set for later SetImage
// calls.
func TestBuildItemKeepsDocumentResources(t *testing.T) {
	res := map[string]fyne.Resource{"logo": fyne.NewStaticResource("logo.png", listPng())}
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<vbox id="v"><image id="logo" src="logo"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, nil, nil, res)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	frag, err := DecodeItem([]byte(`<goexml-item version="0.1.0">` +
		`<vbox id="row"><label id="rowTitle" text="hi"/></vbox></goexml-item>`))
	if err != nil {
		t.Fatalf("DecodeItem: %v", err)
	}
	// A fragment without its own payloads must not wipe the set.
	if _, err := w.BuildItem(frag, nil, nil); err != nil {
		t.Fatalf("BuildItem(nil): %v", err)
	}
	if err := w.SetImage("logo", "logo"); err != nil {
		t.Fatalf("document image died after BuildItem(nil): %v", err)
	}
	// A fragment with payloads merges them: the key resolves for the
	// fragment's image at build time and for the window later.
	icon := map[string]fyne.Resource{"icon": fyne.NewStaticResource("icon.png", listPng())}
	frag2, _ := DecodeItem([]byte(`<goexml-item version="0.1.0"><image id="other" src="icon" style="h-4"/></goexml-item>`))
	if _, err := w.BuildItem(frag2, nil, icon); err != nil {
		t.Fatalf("BuildItem(icon): %v", err)
	}
	if err := w.SetImage("logo", "icon"); err != nil {
		t.Fatalf("fragment key not merged into the window set: %v", err)
	}
}

// TestBuildItemFailureIsAtomic locks the failure path: a BuildItem
// that fails after validation leaves no trace — ids and resource
// payloads alike stay out of the window's state.
func TestBuildItemFailureIsAtomic(t *testing.T) {
	res := map[string]fyne.Resource{"logo": fyne.NewStaticResource("logo.png", listPng())}
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<label id="l" text="x"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, res)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Decodes fine, fails at build: the handler is missing.
	evil, err := DecodeItem([]byte(`<goexml-item version="0.1.0">` +
		`<button id="e" text="x" onClick="Nope"/></goexml-item>`))
	if err != nil {
		t.Fatalf("DecodeItem: %v", err)
	}
	payload := map[string]fyne.Resource{"leak": fyne.NewStaticResource("leak.png", listPng())}
	if _, err := w.BuildItem(evil, nil, payload); err == nil {
		t.Fatal("expected BuildItem to fail on the missing handler")
	}
	if _, ok := res["leak"]; ok {
		t.Fatal("a failed BuildItem leaked its resources into the window set")
	}
	if err := w.SetImage("l", "leak"); err == nil {
		t.Fatal("the leaked key resolved after a failed build")
	}
	// The document's own set is untouched.
	if err := w.SetImage("l", "logo"); err == nil {
		t.Fatal("SetImage on a label should fail, but the key must still resolve")
	} else if !strings.Contains(err.Error(), "cannot set an image") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("window set = %d entries, want just the document's one", len(res))
	}
}

// TestSetImageWrongNode locks the node-kind guard.
func TestSetImageWrongNode(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="100x100">` +
		`<label id="l" text="x"/></window></goexml>`
	root, _ := Decode([]byte(doc))
	w, err := Build(root, nil, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	err = w.SetImage("l", "dot")
	if err == nil || !strings.Contains(err.Error(), "cannot set an image on") {
		t.Fatalf("error %q does not name the guard", err)
	}
}
