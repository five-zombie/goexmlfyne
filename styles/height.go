package styles

import (
	"fmt"
	"strconv"
	"strings"
)

// parseHeightToken handles h-full (fill the parent height) and h-N
// (exact height of N*4 px, N 0-24), e.g. h-16 is a 64 px tall row.
func parseHeightToken(st *Style, tok string) (bool, error) {
	if tok == "h-full" {
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
	st.FixedH = float32(v) * 4
	return true, nil
}
