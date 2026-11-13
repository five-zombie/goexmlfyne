package goexml

import (
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// collectRects walks a tree and gathers canvas rectangles in draw
// order, for asserting the visual layer.
func collectRects(o fyne.CanvasObject, out *[]*canvas.Rectangle) {
	switch t := o.(type) {
	case *fyne.Container:
		for _, c := range t.Objects {
			collectRects(c, out)
		}
	case *canvas.Rectangle:
		*out = append(*out, t)
	}
}

// A styled box gets a visual layer behind its content: the background
// color and border render as canvas rectangles, the box keeps its id,
// and plain nodes stay untouched.
func TestStyledBoxVisualLayer(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="300x200">` +
		`<vbox id="root">` +
		`<vbox id="card" style="bg-[#112233] b-2 b-[#445566] rounded-md p-2">` +
		`<label id="l" text="hi"/></vbox>` +
		`<vbox id="plain"><label id="l2" text="plain"/></vbox>` +
		`</vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var rects []*canvas.Rectangle
	collectRects(w.Content, &rects)
	// Exactly two rectangles: the card's background and its border.
	if len(rects) != 2 {
		t.Fatalf("rectangles = %d, want 2 (card bg + border)", len(rects))
	}
	bg := rects[0]
	if got, ok := bg.FillColor.(color.NRGBA); !ok || got != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}) {
		t.Fatalf("bg fill = %v, want #112233", bg.FillColor)
	}
	if bg.CornerRadius != 8 {
		t.Fatalf("bg radius = %v, want 8 (md)", bg.CornerRadius)
	}
	stroke := rects[1]
	if got, ok := stroke.StrokeColor.(color.NRGBA); !ok || got != (color.NRGBA{R: 0x44, G: 0x55, B: 0x66, A: 255}) {
		t.Fatalf("stroke = %v, want #445566", stroke.StrokeColor)
	}
	if stroke.StrokeWidth != 2 {
		t.Fatalf("stroke width = %v, want 2", stroke.StrokeWidth)
	}
	// The wrapper is transparent to the id space.
	if _, ok := w.Find("l").(*widget.Label); !ok {
		t.Fatalf("Find(l) = %T, want *widget.Label", w.Find("l"))
	}
}

// visible="false" on a styled box hides the whole visual layer.
func TestStyledBoxVisibleFalse(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="card" visible="false" style="bg-[#112233]"><label id="l" text="hi"/></vbox>` +
		`</window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	w, err := Build(root, map[string]func(){}, nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if w.Find("card").Visible() {
		t.Fatal("styled box with visible=false is visible")
	}
	// The wrapper's own hidden flag carries the state; per Fyne
	// semantics the canvas rects inside keep Visible=true (the walk
	// stops at the hidden container), so only the box is asserted.
	var rects []*canvas.Rectangle
	collectRects(w.Content, &rects)
	if len(rects) != 1 {
		t.Fatalf("rectangles = %d, want 1 (the card bg)", len(rects))
	}
}

// Invalid visual combinations are decode errors with node context.
func TestStyledBoxDecodeErrors(t *testing.T) {
	for _, tc := range []struct {
		style, want string
	}{
		{"bg-[#112233] bg-[#445566]", "more than one bg-"},
		{"b-2", "width but no color"},
		{"b-[#ff0000]", "color but no width"},
		{"rounded", "nothing to round"},
		{"rounded-md b-2 b-[#ff0000] bt-[#00ff00]", "uniform"},
	} {
		doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<vbox id="card" style="` + tc.style + `"><label id="l" text="hi"/></vbox>` +
			`</window></goexml>`
		_, err := Decode([]byte(doc))
		if err == nil {
			t.Fatalf("Decode(%q) expected error, got nil", tc.style)
		}
		if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), `id="card"`) {
			t.Fatalf("Decode(%q) error %q misses %q or node context", tc.style, err, tc.want)
		}
	}
}
