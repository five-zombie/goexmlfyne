package goexml

// flagAttrs lists the schema's flag attributes: the ones whose mere
// presence means true. XML has no value-less attributes, so a bare
// flag cannot survive the parser — patchFlags rewrites it to ="true"
// before parsing, so authors can write the natural bare form:
//
//	<vbox-expand id="feed" scrollable />
//
// Any value other than true (including the older empty form) is
// rejected at validation with a migration hint.
var flagAttrs = map[string]bool{
	"scrollable": true,
	"selected":   true,
}

// patchFlags copies data verbatim except for bare flag attributes
// inside start tags, which gain ="true". The scanner tracks quoted
// attribute values wholesale — an attribute value may legally contain
// > or a flag's name, and text and comments must never be touched —
// so a naive substring search is not enough. Bytes that make no sense
// (a raw < inside text, an unterminated tag) are copied through; the
// XML parser reports them as usual.
func patchFlags(data []byte) []byte {
	var out []byte
	i := 0
	for i < len(data) {
		if data[i] != '<' {
			out = append(out, data[i])
			i++
			continue
		}
		// A start tag begins <name; comments <!--, CDATA <![,
		// processing instructions <?, and closing tags </ do not.
		if i+1 >= len(data) || !isNameStart(data[i+1]) {
			out = append(out, data[i])
			i++
			continue
		}
		// Copy the element name, then walk its attributes until the
		// tag closes.
		j := i + 1
		for j < len(data) && isNameByte(data[j]) {
			j++
		}
		out = append(out, data[i:j]...)
		i = j
		for i < len(data) {
			ws := i
			for i < len(data) && isSpace(data[i]) {
				i++
			}
			if i >= len(data) {
				// Truncated tag: copy the rest and let the parser
				// report it.
				out = append(out, data[ws:]...)
				break
			}
			if data[i] == '>' {
				out = append(out, data[ws:i+1]...)
				i++
				break
			}
			if data[i] == '/' && i+1 < len(data) && data[i+1] == '>' {
				out = append(out, data[ws:i+2]...)
				i += 2
				break
			}
			// One attribute name...
			nameStart := i
			for i < len(data) && isNameByte(data[i]) {
				i++
			}
			if i == nameStart {
				// Not a name character (stray = or worse): copy one
				// byte so the parser can place the error.
				out = append(out, data[nameStart:i+1]...)
				i++
				continue
			}
			// ...then optional whitespace and '=' decide bare or
			// valued.
			k := i
			for k < len(data) && isSpace(data[k]) {
				k++
			}
			if k < len(data) && data[k] == '=' {
				// Valued: copy the whole attribute verbatim,
				// including its quoted value (which may hold > or a
				// flag's name).
				v := k + 1
				for v < len(data) && isSpace(data[v]) {
					v++
				}
				if v < len(data) && (data[v] == '"' || data[v] == '\'') {
					q := data[v]
					v++
					for v < len(data) && data[v] != q {
						v++
					}
					if v < len(data) {
						v++ // closing quote
					}
				}
				out = append(out, data[ws:v]...)
				i = v
				continue
			}
			// Bare name: flags are rewritten, anything else is
			// copied and rejected by the parser.
			if flagAttrs[string(data[nameStart:i])] {
				out = append(out, data[ws:i]...)
				out = append(out, `="true"`...)
			} else {
				out = append(out, data[ws:i]...)
			}
		}
	}
	return out
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isNameStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func isNameByte(c byte) bool {
	return isNameStart(c) || c >= '0' && c <= '9' || c == '-' || c == '.' || c == ':'
}
