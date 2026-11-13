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
