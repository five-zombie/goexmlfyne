package styles

// parseFillToken handles the full-width utility: fill the parent cross
// axis without taking main-axis space.
func parseFillToken(st *Style, tok string) (bool, error) {
	if tok != "full-width" {
		return false, nil
	}
	st.Fill = true
	return true, nil
}
