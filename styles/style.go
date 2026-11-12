package styles

import (
	"fmt"
	"sort"
	"strings"
)

// Style is the parsed value of a style attribute.
type Style struct {
	// WFull fills the parent width.
	WFull bool
	// HFull fills the parent height.
	HFull bool
	// FixedH is an exact height in pixels, or 0 for none.
	FixedH float32
	// Pad is the inner spacing per side.
	Pad Edges
	// Margin is the outer spacing per side.
	Margin Edges
	// Vis is the parsed visual layer: background, border and rounded
	// corners. Zero when no visual token appeared.
	Vis Visual

	// hasW and hasH record whether any w-*/h-* utility appeared,
	// so duplicates are rejected even when the value parses to zero
	// (h-0 leaves FixedH at 0). hasFixed marks an h-N utility in
	// particular, including h-0, which is a real zero height.
	hasW, hasH, hasFixed bool
}

// HasFixedH reports whether an exact height utility (h-N, including
// h-0) is present. h-full does not count.
func (s Style) HasFixedH() bool { return s.hasFixed }

// spacingAssign is one parsed token waiting for tier-ordered application.
type spacingAssign struct {
	tier  int
	apply func()
}

// ParseStyle parses and validates a style attribute.
// Unknown utilities and out-of-range spacing are errors.
func ParseStyle(s string) (Style, error) {
	var st Style
	var assigns []spacingAssign
	for _, tok := range strings.Fields(s) {
		if ok, err := parseWidthToken(&st, tok); err != nil {
			return Style{}, err
		} else if ok {
			continue
		}
		if ok, err := parseHeightToken(&st, tok); err != nil {
			return Style{}, err
		} else if ok {
			continue
		}
		if tier, apply, ok, err := parsePaddingAssign(&st.Pad, tok); err != nil {
			return Style{}, err
		} else if ok {
			assigns = append(assigns, spacingAssign{tier, apply})
			continue
		}
		if tier, apply, ok, err := parseMarginAssign(&st.Margin, tok); err != nil {
			return Style{}, err
		} else if ok {
			assigns = append(assigns, spacingAssign{tier, apply})
			continue
		}
		if ok, err := parseBGToken(&st.Vis, tok); err != nil {
			return Style{}, err
		} else if ok {
			continue
		}
		if tier, apply, ok, err := parseBorderToken(&st.Vis, tok); err != nil {
			return Style{}, err
		} else if ok {
			assigns = append(assigns, spacingAssign{tier, apply})
			continue
		}
		if tier, apply, ok, err := parseRadiusToken(&st.Vis, tok); err != nil {
			return Style{}, err
		} else if ok {
			assigns = append(assigns, spacingAssign{tier, apply})
			continue
		}
		return Style{}, fmt.Errorf("unknown style %q", tok)
	}
	sortAssigns(assigns)
	for _, a := range assigns {
		a.apply()
	}
	// The visual layer is validated as a whole: pairs and rounding
	// rules see the merged result, not single tokens.
	if st.Vis.Active() {
		if err := st.Vis.validate(); err != nil {
			return Style{}, err
		}
	}
	return st, nil
}

// sortAssigns orders assignments by specificity tier, keeping token
// order within a tier.
func sortAssigns(assigns []spacingAssign) {
	sort.SliceStable(assigns, func(i, j int) bool { return assigns[i].tier < assigns[j].tier })
}
