package styles

// parseWidthToken handles the w-full utility: fill the parent width.
func parseWidthToken(st *Style, tok string) bool {
	if tok == "w-full" {
		st.WFull = true
		return true
	}
	return false
}
