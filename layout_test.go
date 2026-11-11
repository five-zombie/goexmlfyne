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

func TestBoxMinSize(t *testing.T) {
	l := newBoxLayout(true, 8, 4, "start", []bool{false, false}, []bool{false, false})
	got := l.MinSize([]fyne.CanvasObject{newStub(100, 20), newStub(50, 30)})
	want := fyne.NewSize(116, 70)
	if got != want {
		t.Fatalf("MinSize = %v, want %v", got, want)
	}
}

func TestBoxLayoutExpand(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	l := newBoxLayout(true, 8, 4, "start", []bool{false, true}, []bool{false, false})
	size := fyne.NewSize(116, 170)
	l.Layout([]fyne.CanvasObject{a, b}, size)
	// Content height is 170-16=154. Children take 20+30+4=54,
	// so the expanding child grows by 100.
	if b.size.Height != 130 {
		t.Fatalf("expanding height = %v, want 130", b.size.Height)
	}
	if b.size.Width != 100 {
		t.Fatalf("expanding width = %v, want 100", b.size.Width)
	}
	if a.pos.Y != 8 || b.pos.Y != 32 {
		t.Fatalf("positions = %v and %v, want Y 8 and 32", a.pos, b.pos)
	}
}

func TestBoxLayoutFill(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	l := newBoxLayout(true, 8, 4, "start", []bool{false, false}, []bool{true, false})
	l.Layout([]fyne.CanvasObject{a, b}, fyne.NewSize(116, 170))
	if a.size.Width != 100 || a.size.Height != 20 {
		t.Fatalf("fill child size = %v, want 100x20", a.size)
	}
	if b.pos.Y != 32 {
		t.Fatalf("second child Y = %v, want 32", b.pos.Y)
	}
}

func TestBoxLayoutSkipsHidden(t *testing.T) {
	a, b := newStub(100, 20), newStub(50, 30)
	a.Hide()
	l := newBoxLayout(true, 0, 4, "start", []bool{false, false}, []bool{false, false})
	l.Layout([]fyne.CanvasObject{a, b}, fyne.NewSize(50, 30))
	if b.pos.Y != 0 {
		t.Fatalf("visible child Y = %v, want 0", b.pos.Y)
	}
}
