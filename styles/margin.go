package styles

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
)

// marginRules maps m-family prefixes to side masks, specific first.
var marginRules = []struct {
	prefix string
	mask   sideMask
}{
	{"mx-", sideX}, {"my-", sideY},
	{"mt-", sideTop}, {"mr-", sideRight}, {"mb-", sideBottom}, {"ml-", sideLeft},
	{"m-", sideAll},
}

// parseMarginAssign resolves one m-family token into a tiered assignment:
// m-N all sides, mx-/my- one axis, mt-/mr-/mb-/ml- one side, N 0-4.
func parseMarginAssign(dst *Edges, tok string) (tier int, apply func(), handled bool, err error) {
	for _, r := range marginRules {
		if !strings.HasPrefix(tok, r.prefix) {
			continue
		}
		v, err := strconv.Atoi(tok[len(r.prefix):])
		if err != nil || v < 0 || v > 4 {
			return 0, nil, true, fmt.Errorf("bad margin %q, want %s0 to %s4", tok, r.prefix, r.prefix)
		}
		px, mask := float32(v)*4, r.mask
		return spacingTier(mask), func() { dst.set(mask, px) }, true, nil
	}
	return 0, nil, false, nil
}

// NewMargin wraps obj with pad pixels of outer spacing on every side.
//
// Split of responsibilities inside this package: padding.go owns the
// geometry (how the spacing is laid out), this file owns the policy
// (when a styled element gets a wrapper and how child state propagates
// to it). A hidden child produces a hidden wrapper so parent box
// layouts collapse the gap instead of leaving an empty padded slot.
func NewMargin(pad Edges, obj fyne.CanvasObject) fyne.CanvasObject {
	w := NewPad(pad, obj)
	if !obj.Visible() {
		w.Hide()
	}
	return w
}
