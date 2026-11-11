package styles

import (
	"fmt"
	"sort"
	"strings"
)

// Style is the parsed value of a style attribute.
type Style struct {
	// Fill reports whether the element fills the parent cross axis.
	Fill bool
	// Pad is the inner spacing per side.
	Pad Edges
	// Margin is the outer spacing per side.
	Margin Edges
}

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
		if ok, _ := parseFillToken(&st, tok); ok {
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
		return Style{}, fmt.Errorf("unknown style %q", tok)
	}
	sortAssigns(assigns)
	for _, a := range assigns {
		a.apply()
	}
	return st, nil
}

// sortAssigns orders assignments by specificity tier, keeping token
// order within a tier.
func sortAssigns(assigns []spacingAssign) {
	sort.SliceStable(assigns, func(i, j int) bool { return assigns[i].tier < assigns[j].tier })
}
