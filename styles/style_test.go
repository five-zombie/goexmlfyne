package styles

import (
	"strings"
	"testing"
)

func TestParseStyle(t *testing.T) {
	st, err := ParseStyle("w-full p-2")
	if err != nil {
		t.Fatalf("ParseStyle returned error: %v", err)
	}
	if !st.WFull || st.Pad != UniformEdges(8) {
		t.Fatalf("ParseStyle = %+v, want WFull and Pad 8", st)
	}
	st, err = ParseStyle("")
	if err != nil {
		t.Fatalf("ParseStyle empty returned error: %v", err)
	}
	if st.WFull || st.HFull || st.FixedH != 0 || !st.Pad.IsZero() || !st.Margin.IsZero() {
		t.Fatalf("ParseStyle empty = %+v, want zero", st)
	}
	for _, s := range []string{"wide", "p-9", "p-x", "p-", "h-99", "h-x", "h-"} {
		if _, err := ParseStyle(s); err == nil {
			t.Fatalf("ParseStyle(%q) expected error, got nil", s)
		} else if !strings.Contains(err.Error(), s) {
			t.Fatalf("error %q does not name %q", err, s)
		}
	}
	st, err = ParseStyle("m-2")
	if err != nil {
		t.Fatalf("ParseStyle m-2 returned error: %v", err)
	}
	if st.Margin != UniformEdges(8) || st.WFull || !st.Pad.IsZero() {
		t.Fatalf("ParseStyle m-2 = %+v, want only Margin 8", st)
	}
	st, err = ParseStyle("h-full h-2")
	if err != nil {
		t.Fatalf("ParseStyle h-full h-2 returned error: %v", err)
	}
	if !st.HFull || st.FixedH != 8 {
		t.Fatalf("ParseStyle = %+v, want HFull and FixedH 8", st)
	}
	st, err = ParseStyle("p-2 pt-4 mx-1 ml-3")
	if err != nil {
		t.Fatalf("ParseStyle directional returned error: %v", err)
	}
	wantPad := Edges{Top: 16, Right: 8, Bottom: 8, Left: 8}
	if st.Pad != wantPad {
		t.Fatalf("ParseStyle pad = %+v, want %+v", st.Pad, wantPad)
	}
	wantMargin := Edges{Top: 0, Right: 4, Bottom: 0, Left: 12}
	if st.Margin != wantMargin {
		t.Fatalf("ParseStyle margin = %+v, want %+v", st.Margin, wantMargin)
	}
	// Specificity wins regardless of order.
	rev, err := ParseStyle("pt-4 p-2")
	if err != nil {
		t.Fatalf("ParseStyle reversed returned error: %v", err)
	}
	if rev.Pad.Top != 16 {
		t.Fatalf("ParseStyle reversed top = %v, want 16", rev.Pad.Top)
	}
}
