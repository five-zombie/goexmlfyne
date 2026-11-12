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
