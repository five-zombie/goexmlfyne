package styles

import (
	"strings"
	"testing"
)

// TestParseVisual covers the bg/border/rounded token matrix: values,
// specificity tiers and every rejection path. Token-level errors name
// the offending token; combination-level (validate) errors name the
// offending side or rule instead.
func TestParseVisual(t *testing.T) {
	// Background alone parses.
	st, err := ParseStyle("bg-[#112233]")
	if err != nil {
		t.Fatalf("ParseStyle bg returned error: %v", err)
	}
	if !st.Vis.HasBG || st.Vis.BG != (Color{0x11, 0x22, 0x33}) {
		t.Fatalf("ParseStyle bg = %+v, want #112233", st.Vis)
	}
	// Full combo: color and width on all sides, all corners rounded.
	st, err = ParseStyle("bg-[#112233] b-2 b-[#445566] rounded-md")
	if err != nil {
		t.Fatalf("ParseStyle combo returned error: %v", err)
	}
	if st.Vis.BorderW != UniformEdges(2) {
		t.Fatalf("border width = %+v, want 2 on all sides", st.Vis.BorderW)
	}
	wantC := ColorEdges{Color{0x44, 0x55, 0x66}, Color{0x44, 0x55, 0x66}, Color{0x44, 0x55, 0x66}, Color{0x44, 0x55, 0x66}}
	if st.Vis.C != wantC {
		t.Fatalf("border colors = %+v, want #445566 everywhere", st.Vis.C)
	}
	if st.Vis.Radius != (Corners{8, 8, 8, 8}) {
		t.Fatalf("radius = %+v, want md (8) everywhere", st.Vis.Radius)
	}
	// Specificity: side beats all regardless of order, corners beat sides.
	st, err = ParseStyle("b-1 bt-3 b-[#ff0000] bt-[#00ff00]")
	if err != nil {
		t.Fatalf("ParseStyle border sides returned error: %v", err)
	}
	if st.Vis.BorderW.Top != 3 || st.Vis.BorderW.Right != 1 {
		t.Fatalf("border widths = %+v, want top 3 rest 1", st.Vis.BorderW)
	}
	if st.Vis.C.Top != (Color{0, 255, 0}) || st.Vis.C.Right != (Color{255, 0, 0}) {
		t.Fatalf("border colors = %+v, want top green rest red", st.Vis.C)
	}
	st, err = ParseStyle("bg-[#000000] rounded-xl rounded-t-md rounded-bl-12")
	if err != nil {
		t.Fatalf("ParseStyle rounded corners returned error: %v", err)
	}
	if st.Vis.Radius != (Corners{TL: 8, TR: 8, BR: 16, BL: 12}) {
		t.Fatalf("radius = %+v, want per-corner merge", st.Vis.Radius)
	}
	// Free pixel size and bare default.
	st, err = ParseStyle("bg-[#000000] rounded-12")
	if err != nil {
		t.Fatalf("ParseStyle rounded-12 returned error: %v", err)
	}
	if st.Vis.Radius != (Corners{12, 12, 12, 12}) {
		t.Fatalf("radius = %+v, want 12 everywhere", st.Vis.Radius)
	}
	st, err = ParseStyle("bg-[#000000] rounded")
	if err != nil {
		t.Fatalf("ParseStyle rounded returned error: %v", err)
	}
	if st.Vis.Radius != (Corners{4, 4, 4, 4}) {
		t.Fatalf("radius = %+v, want default 4", st.Vis.Radius)
	}
	// Token-level errors: the message names the offending token.
	for _, s := range []string{
		"bg-x",           // no color value
		"bg-#112233",     // missing brackets
		"bg-[#1122]",     // short hex
		"bg-[#11223g]",   // bad hex digit
		"bg-[#ff0000",    // unclosed bracket
		"b-x",            // not a number
		"b-17",           // out of range
		"b-[#f0f]",       // short hex in border
		"b-[#ff0000",     // unclosed bracket
		"rounded-0",      // dead: zero radius
		"rounded-lg",     // unknown named size
		"rounded-t",      // side without explicit size
		"rounded-t-b-12", // too many parts
		"rounded-x-md",   // unknown edge
	} {
		_, err := ParseStyle(s)
		if err == nil {
			t.Fatalf("ParseStyle(%q) expected error, got nil", s)
		} else if !strings.Contains(err.Error(), s) {
			t.Fatalf("ParseStyle(%q) error %q does not name the token", s, err)
		}
	}
	// Combination-level errors from validate: the message names the
	// offending side or rule, since the offender is the merge result.
	for _, tc := range []struct {
		style, want string
	}{
		{"bg-[#112233] bg-[#445566]", "more than one bg-"},
		{"b-2", "width but no color"},
		{"b-[#ff0000]", "color but no width"},
		{"b-2 bt-[#00ff00]", "width but no color"},
		{"b-2 b-[#ff0000] bt-0", "zero width"},
		{"rounded", "nothing to round"},
		{"rounded-md b-2 b-[#ff0000] bt-[#00ff00]", "uniform"},
		{"bg-[#000000] rounded-md b-2 bt-3 b-[#ff0000]", "uniform"},
	} {
		_, err := ParseStyle(tc.style)
		if err == nil {
			t.Fatalf("ParseStyle(%q) expected error, got nil", tc.style)
		} else if !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("ParseStyle(%q) error %q does not contain %q", tc.style, err, tc.want)
		}
	}
	// Rounded with a uniform border and a background is fine.
	st, err = ParseStyle("bg-[#112233] rounded-md b-2 b-[#445566]")
	if err != nil {
		t.Fatalf("ParseStyle uniform rounded returned error: %v", err)
	}
	if st.Vis.Radius.Max() != 8 {
		t.Fatalf("radius max = %v, want 8", st.Vis.Radius.Max())
	}
	// Plain spacing tokens keep working beside visual ones.
	st, err = ParseStyle("bg-[#112233] p-2 pt-4 m-1")
	if err != nil {
		t.Fatalf("ParseStyle mixed returned error: %v", err)
	}
	if st.Pad.Top != 16 || st.Margin != UniformEdges(4) {
		t.Fatalf("mixed spacing = pad %+v margin %+v", st.Pad, st.Margin)
	}
}
