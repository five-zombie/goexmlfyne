package goexml

import (
	"testing"

	"fyne.io/fyne/v2"
)

// stubObject is a fixed-size CanvasObject for layout math tests.
type stubObject struct {
	min   fyne.Size
	pos   fyne.Position
	size  fyne.Size
	shown bool
}

func newStub(w, h float32) *stubObject {
	return &stubObject{min: fyne.NewSize(w, h), shown: true}
}

func (s *stubObject) MinSize() fyne.Size      { return s.min }
func (s *stubObject) Resize(size fyne.Size)   { s.size = size }
func (s *stubObject) Position() fyne.Position { return s.pos }
func (s *stubObject) Move(pos fyne.Position)  { s.pos = pos }
func (s *stubObject) Size() fyne.Size         { return s.size }
func (s *stubObject) Visible() bool           { return s.shown }
func (s *stubObject) Show()                   { s.shown = true }
func (s *stubObject) Hide()                   { s.shown = false }
func (s *stubObject) Refresh()                {}

func hints(w, h []bool, fixed []float32) []childHint {
	var out []childHint
	for i := range w {
		var f float32
		set := i < len(fixed)
		if set {
			f = fixed[i]
		}
		out = append(out, childHint{wFull: w[i], hFull: h[i], fixedH: f, fixedSet: set})
	}
	return out
}

func TestBoxMinSize(t *testing.T) {
	l := newBoxLayout(true, 4, "start", "start", hints([]bool{false, false}, []bool{false, false}, nil))
	got := l.MinSize([]fyne.CanvasObject{newStub(100, 20), newStub(50, 30)})
	want := fyne.NewSize(100, 54)
	if got != want {
		t.Fatalf("MinSize = %v, want %v", got, want)
	}
}

func TestBoxLayoutExpand(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	l := newBoxLayout(true, 4, "start", "start", hints([]bool{true, true}, []bool{false, true}, nil))
	size := fyne.NewSize(116, 170)
	l.Layout([]fyne.CanvasObject{a, b}, size)
	// Children take 20+30+4=54, so the expanding child grows by 116.
	if b.size.Height != 146 {
		t.Fatalf("expanding height = %v, want 146", b.size.Height)
	}
	if b.size.Width != 116 {
		t.Fatalf("expanding width = %v, want 116 (full cross axis)", b.size.Width)
	}
	if a.pos.Y != 0 || b.pos.Y != 24 {
		t.Fatalf("positions = %v and %v, want Y 0 and 24", a.pos, b.pos)
	}
}

func TestBoxLayoutFill(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	l := newBoxLayout(true, 4, "start", "start", hints([]bool{true, false}, []bool{false, false}, nil))
	l.Layout([]fyne.CanvasObject{a, b}, fyne.NewSize(116, 170))
	if a.size.Width != 116 || a.size.Height != 20 {
		t.Fatalf("fill child size = %v, want 116x20 (full cross axis)", a.size)
	}
	if b.pos.Y != 24 {
		t.Fatalf("second child Y = %v, want 24", b.pos.Y)
	}
}

func TestBoxLayoutFixedH(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	l := newBoxLayout(true, 4, "start", "start", hints([]bool{false, false}, []bool{false, false}, []float32{40}))
	if got := l.MinSize([]fyne.CanvasObject{a, b}); got != fyne.NewSize(100, 74) {
		t.Fatalf("MinSize = %v, want 100x74", got)
	}
	l.Layout([]fyne.CanvasObject{a, b}, fyne.NewSize(116, 74))
	if a.size.Height != 40 {
		t.Fatalf("fixed height = %v, want 40", a.size.Height)
	}
	if b.pos.Y != 44 {
		t.Fatalf("second child Y = %v, want 44", b.pos.Y)
	}
}

// A zero fixed height collapses the child but the spacing gutter and
// the rest of the layout keep flowing around it.
func TestBoxLayoutFixedHZero(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	l := newBoxLayout(true, 4, "start", "start", hints([]bool{false, false}, []bool{false, false}, []float32{0}))
	if got := l.MinSize([]fyne.CanvasObject{a, b}); got != fyne.NewSize(100, 34) {
		t.Fatalf("MinSize = %v, want 100x34 (0 + 30 + spacing 4)", got)
	}
	l.Layout([]fyne.CanvasObject{a, b}, fyne.NewSize(116, 70))
	if a.size.Height != 0 {
		t.Fatalf("zero fixed height = %v, want 0", a.size.Height)
	}
	if b.pos.Y != 4 {
		t.Fatalf("second child Y = %v, want 4 (0 + spacing 4)", b.pos.Y)
	}
}

func TestGridLayoutColumns(t *testing.T) {
	objs := []fyne.CanvasObject{
		newStub(100, 20), newStub(100, 20), newStub(100, 20),
		newStub(100, 20), newStub(100, 20),
	}
	l := newGridLayout(4, 2, 4, nil)
	if got := l.MinSize(objs); got != fyne.NewSize(204, 68) {
		t.Fatalf("MinSize = %v, want 204x68 (2 columns assumed)", got)
	}
	l.Layout(objs, fyne.NewSize(368, 400))
	cellW := float32(120)
	for i, o := range objs[:3] {
		if o.Size().Width != cellW {
			t.Fatalf("child %d width = %v, want %v", i, o.Size().Width, cellW)
		}
		if o.Position().Y != 0 {
			t.Fatalf("child %d Y = %v, want 0 (first row)", i, o.Position().Y)
		}
	}
	if objs[3].Position().Y != 202 {
		t.Fatalf("child 3 Y = %v, want 202 (second row)", objs[3].Position().Y)
	}
	if objs[3].Size().Height != 198 || objs[0].Size().Height != 198 {
		t.Fatalf("row heights = %v and %v, want 198 each", objs[0].Size().Height, objs[3].Size().Height)
	}
}

func TestBoxLayoutJustify(t *testing.T) {
	mkib := func() []fyne.CanvasObject { return []fyne.CanvasObject{newStub(100, 20), newStub(50, 30)} }
	// total = 150 + 4 spacing = 154; container 300 wide → extra 146.
	cases := []struct {
		mode string
		x0   float32
		x1   float32
	}{
		{"start", 0, 104},
		{"center", 73, 177},
		{"end", 146, 250},
		{"space-between", 0, 250},
		{"space-around", 36.5, 213.5},
		{"space-evenly", float32(146) / 3, float32(146)/3 + 100 + (4 + float32(146)/3)},
	}
	for _, c := range cases {
		objs := mkib()
		l := newBoxLayout(false, 4, "start", c.mode, hints([]bool{false, false}, []bool{false, false}, nil))
		l.Layout(objs, fyne.NewSize(300, 60))
		if objs[0].Position().X != c.x0 || objs[1].Position().X != c.x1 {
			t.Fatalf("justify %s: positions = %v and %v, want %v and %v",
				c.mode, objs[0].Position().X, objs[1].Position().X, c.x0, c.x1)
		}
	}
}

func TestBoxLayoutExpandBeatsJustify(t *testing.T) {
	objs := []fyne.CanvasObject{newStub(100, 20), newStub(50, 30)}
	l := newBoxLayout(false, 4, "start", "end", hints([]bool{false, true}, []bool{false, false}, nil))
	l.Layout(objs, fyne.NewSize(300, 60))
	if objs[0].Position().X != 0 || objs[1].Position().X != 104 {
		t.Fatalf("positions = %v and %v, want 0 and 104 (expand eats extra first)",
			objs[0].Position().X, objs[1].Position().X)
	}
}

func TestBoxLayoutSkipsHidden(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	a.Hide()
	l := newBoxLayout(true, 4, "start", "start", hints([]bool{false, false}, []bool{false, false}, nil))
	l.Layout([]fyne.CanvasObject{a, b}, fyne.NewSize(50, 30))
	if b.pos.Y != 0 {
		t.Fatalf("visible child Y = %v, want 0", b.pos.Y)
	}
}
