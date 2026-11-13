package styles

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// TestNewVisualLayerOrder covers the wrapper: layer order (background
// under border under content), min-size passthrough, per-side edge
// geometry and radius clamping.
func TestNewVisualLayerOrder(t *testing.T) {
	st, err := ParseStyle("bg-[#112233] b-2 b-[#00ff00] rounded-40")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	v := st.Vis
	content := canvas.NewRectangle(color.NRGBA{R: 9, G: 9, B: 9, A: 255})
	w := NewVisual(v, content)
	c, ok := w.(*fyne.Container)
	if !ok {
		t.Fatalf("NewVisual = %T, want *fyne.Container", w)
	}
	// bg, one stroked border rect (uniform + rounded), content.
	if len(c.Objects) != 3 {
		t.Fatalf("layers = %d, want 3 (bg, stroke, content)", len(c.Objects))
	}
	bg := c.Objects[0].(*canvas.Rectangle)
	if got := bg.FillColor.(color.NRGBA); got != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}) {
		t.Fatalf("bg fill = %v, want #112233", got)
	}
	if bg.CornerRadius != 40 {
		t.Fatalf("bg radius = %v, want unclamped 40", bg.CornerRadius)
	}
	stroke := c.Objects[1].(*canvas.Rectangle)
	if got := stroke.StrokeColor.(color.NRGBA); got != (color.NRGBA{R: 0, G: 255, B: 0, A: 255}) {
		t.Fatalf("stroke color = %v, want green", got)
	}
	if stroke.StrokeWidth != 2 {
		t.Fatalf("stroke width = %v, want 2", stroke.StrokeWidth)
	}
	if c.Objects[2] != content {
		t.Fatal("content is not the top layer")
	}
	// Min size passes through: the layer is inset.
	if got := w.MinSize(); got != content.MinSize() {
		t.Fatalf("MinSize = %v, want content %v", got, content.MinSize())
	}
	// Layout fills the whole area behind the content and clamps the
	// radius (40) to half the smaller dimension (25).
	w.Resize(fyne.NewSize(100, 50))
	if bg.Size().Width != 100 || bg.Size().Height != 50 {
		t.Fatalf("bg size = %v, want 100x50", bg.Size())
	}
	if content.Position() != (fyne.Position{}) || content.Size() != (fyne.NewSize(100, 50)) {
		t.Fatalf("content = %v %v, want origin and full size", content.Position(), content.Size())
	}
	if bg.CornerRadius != 25 {
		t.Fatalf("clamped radius = %v, want 25", bg.CornerRadius)
	}
}

// TestNewVisualPerSideEdges covers the four thin edge rectangles for a
// non-uniform border without rounding: each edge hugs its side, and
// undrawn sides are zero-size.
func TestNewVisualPerSideEdges(t *testing.T) {
	st, err := ParseStyle("bg-[#000000] bt-2 bt-[#ff0000] bl-4 bl-[#0000ff]")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	v := st.Vis
	content := canvas.NewRectangle(color.NRGBA{A: 255})
	w := NewVisual(v, content)
	c := w.(*fyne.Container)
	// bg, 4 edge rects, content.
	if len(c.Objects) != 6 {
		t.Fatalf("layers = %d, want 6 (bg, 4 edges, content)", len(c.Objects))
	}
	w.Resize(fyne.NewSize(100, 50))
	top := c.Objects[1].(*canvas.Rectangle)
	left := c.Objects[4].(*canvas.Rectangle)
	right := c.Objects[2].(*canvas.Rectangle)
	bottom := c.Objects[3].(*canvas.Rectangle)
	if top.Size() != (fyne.NewSize(100, 2)) || top.Position() != (fyne.Position{}) {
		t.Fatalf("top edge = %v @ %v, want 100x2 @ origin", top.Size(), top.Position())
	}
	if left.Size() != (fyne.NewSize(4, 50)) {
		t.Fatalf("left edge = %v, want 4x50", left.Size())
	}
	if right.Size() != (fyne.NewSize(0, 50)) || bottom.Size() != (fyne.NewSize(100, 0)) {
		t.Fatalf("undrawn edges = %v %v, want zero thickness", right.Size(), bottom.Size())
	}
}

// TestNewVisualStrokeGeometry locks the stroke inset fix: Fyne draws
// a stroke centered on its path, so at plain bounds half the thickness
// hangs outside the element. The layout insets the stroke rect by half
// the width so the stroke's outer edge sits on the bounds — the same
// geometry the per-side edges have — and a border thicker than the
// element clamps to an empty rect instead of mirroring.
func TestNewVisualStrokeGeometry(t *testing.T) {
	st, err := ParseStyle("b-4 b-[#00ff00] rounded-md")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	content := canvas.NewRectangle(color.NRGBA{A: 255})
	w := NewVisual(st.Vis, content)
	c := w.(*fyne.Container)
	// No bg: stroke rect first, content last.
	if len(c.Objects) != 2 {
		t.Fatalf("layers = %d, want 2 (stroke, content)", len(c.Objects))
	}
	stroke := c.Objects[0].(*canvas.Rectangle)
	w.Resize(fyne.NewSize(100, 60))
	// Inset by half of the 4 px width; outer edge on the bounds.
	if stroke.Position() != (fyne.NewPos(2, 2)) {
		t.Fatalf("stroke pos = %v, want (2,2)", stroke.Position())
	}
	if stroke.Size() != (fyne.NewSize(96, 56)) {
		t.Fatalf("stroke size = %v, want 96x56", stroke.Size())
	}
	// Radius shrinks by half the width: the inset rect's corner arcs
	// stay concentric with the style's radius, so the stroke's outer
	// edge follows the silhouette (8 px radius at the bounds).
	if stroke.CornerRadius != 6 {
		t.Fatalf("stroke radius = %v, want 8-4/2 = 6", stroke.CornerRadius)
	}
	if stroke.TopLeftCornerRadius != 0 || stroke.BottomRightCornerRadius != 0 {
		t.Fatalf("per-corner radius overrides = %v/%v, want zeroed by the uniform path",
			stroke.TopLeftCornerRadius, stroke.BottomRightCornerRadius)
	}

	// Concentricity against a real background: the bg keeps the style
	// radius (clamped), the stroke path's radius is smaller by half
	// the width, and the inset matches — outer edge on the silhouette.
	bgSt, err := ParseStyle("bg-[#112233] b-4 b-[#00ff00] rounded-md")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	w4 := NewVisual(bgSt.Vis, canvas.NewRectangle(color.NRGBA{A: 255}))
	c4 := w4.(*fyne.Container)
	w4.Resize(fyne.NewSize(100, 60))
	bg4 := c4.Objects[0].(*canvas.Rectangle)
	stroke4 := c4.Objects[1].(*canvas.Rectangle)
	if bg4.CornerRadius != 8 || stroke4.CornerRadius != 6 {
		t.Fatalf("concentric radii = bg %v / stroke %v, want 8 / 8-4/2=6",
			bg4.CornerRadius, stroke4.CornerRadius)
	}
	if stroke4.Position() != (fyne.NewPos(2, 2)) {
		t.Fatalf("stroke pos = %v, want inset (2,2)", stroke4.Position())
	}

	// The per-side path keeps its own geometry: edges hug the bounds.
	ps, err := ParseStyle("bt-2 bt-[#ff0000]")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	w2 := NewVisual(ps.Vis, canvas.NewRectangle(color.NRGBA{A: 255}))
	c2 := w2.(*fyne.Container)
	w2.Resize(fyne.NewSize(100, 60))
	top := c2.Objects[0].(*canvas.Rectangle)
	if top.Position() != (fyne.Position{}) || top.Size() != (fyne.NewSize(100, 2)) {
		t.Fatalf("top edge = %v @ %v, want 100x2 @ origin", top.Size(), top.Position())
	}

	// A border thicker than the element clamps to an empty rect
	// (rounded style takes the stroke path).
	th, err := ParseStyle("b-8 b-[#00ff00] rounded-md")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	w3 := NewVisual(th.Vis, canvas.NewRectangle(color.NRGBA{A: 255}))
	c3 := w3.(*fyne.Container)
	w3.Resize(fyne.NewSize(6, 6))
	tiny := c3.Objects[0].(*canvas.Rectangle)
	if tiny.Size() != (fyne.NewSize(0, 0)) || tiny.Position() != (fyne.NewPos(4, 4)) {
		t.Fatalf("inverted stroke = %v @ %v, want 0x0 @ (4,4)", tiny.Size(), tiny.Position())
	}
}

// TestNewVisualHiddenContent covers the visibility propagation: a
// hidden content produces a hidden wrapper so parent layouts collapse
// the slot.
func TestNewVisualHiddenContent(t *testing.T) {
	st, err := ParseStyle("bg-[#000000]")
	if err != nil {
		t.Fatalf("ParseStyle: %v", err)
	}
	content := canvas.NewRectangle(color.NRGBA{A: 255})
	w := NewVisual(st.Vis, content)
	if !w.Visible() {
		t.Fatal("visible content produced a hidden wrapper")
	}
	content.Hide()
	w = NewVisual(st.Vis, content)
	if w.Visible() {
		t.Fatal("hidden content produced a visible wrapper")
	}
}
