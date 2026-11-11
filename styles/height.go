package styles

import (
	"fmt"
	"strconv"
	"strings"
)

// parseHeightToken handles h-full (fill the parent height) and h-N
// (exact height of N*4 px, N 0-24), e.g. h-16 is a 64 px tall row.
// h-0 is a real zero height: an invisible element that still takes
// part in layout through its margin. A tag carries at most one h-*
// utility, so a second — including the h-full plus h-N mix — is an error.
func parseHeightToken(st *Style, tok string) (bool, error) {
	if tok == "h-full" || strings.HasPrefix(tok, "h-") {
		if st.hasH {
			return true, fmt.Errorf("style has more than one h-*: %q", tok)
		}
	}
	if tok == "h-full" {
		st.hasH = true
		st.HFull = true
		return true, nil
	}
	if !strings.HasPrefix(tok, "h-") {
		return false, nil
	}
	v, err := strconv.Atoi(tok[2:])
	if err != nil || v < 0 || v > 24 {
		return true, fmt.Errorf("bad height %q, want h-0 to h-24", tok)
	}
	st.hasH = true
	st.hasFixed = true
	st.FixedH = float32(v) * 4
	return true, nil
}
