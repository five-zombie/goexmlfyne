package styles

import "fmt"

// parseWidthToken handles the w-full utility: fill the parent width.
// A tag carries at most one w-* utility, so a second is an error.
func parseWidthToken(st *Style, tok string) (bool, error) {
	if tok == "w-full" {
		if st.hasW {
			return true, fmt.Errorf("style has more than one w-*: %q", tok)
		}
		st.hasW = true
		st.WFull = true
		return true, nil
	}
	return false, nil
}
