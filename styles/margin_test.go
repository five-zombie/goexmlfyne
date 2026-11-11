package styles

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestNewMargin(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	lbl := widget.NewLabel("Hi")
	w := NewMargin(UniformEdges(8), lbl)
	if !w.Visible() {
		t.Fatal("wrapper of a visible child is hidden")
	}
	min := lbl.MinSize()
	if got := w.MinSize(); got != fyne.NewSize(min.Width+16, min.Height+16) {
		t.Fatalf("wrapper MinSize = %v, want label min + 16 px", got)
	}
	box, ok := w.(*fyne.Container)
	if !ok {
		t.Fatalf("wrapper is %T, want *fyne.Container", w)
	}
	if box.Objects[0] != lbl {
		t.Fatal("wrapper does not hold the original object")
	}

	hidden := widget.NewLabel("gone")
	hidden.Hide()
	if w := NewMargin(UniformEdges(8), hidden); w.Visible() {
		t.Fatal("wrapper of a hidden child is visible")
	}
}
