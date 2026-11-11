package goexml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/five-zombie/goexmlfyne/styles"
)

// idPattern is the id contract from the standard: letters, digits and
// underscore only, so ids stay referenceable in handlers and errors.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// Decode parses a goexml 0.1.0 document and validates it against the
// schema. It returns the root node. Every error carries the 1-based
// source line of the offending element.
func Decode(data []byte) (*Node, error) {
	// A UTF-8 BOM is common from Windows editors; strip it instead of
	// misreporting it as text outside the root element.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	d := xml.NewDecoder(bytes.NewReader(data))
	lt := newLineTracker(data)
	var root *Node
	for root == nil {
		off := d.InputOffset()
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("goexml: line %d: document has no root element", lt.at(off))
			}
			return nil, fmt.Errorf("goexml: line %d: %v", lt.at(off), err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			r, err := parseElement(lt, d, t, off)
			if err != nil {
				return nil, err
			}
			if err := validate(r); err != nil {
				return nil, err
			}
			root = r
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return nil, fmt.Errorf("goexml: line %d: text outside root element", lt.at(off))
			}
		case xml.Comment, xml.ProcInst, xml.Directive:
			continue
		default:
			return nil, fmt.Errorf("goexml: line %d: unexpected token", lt.at(off))
		}
	}
	// Anything after the root must be trailing whitespace or comments;
	// a second element or stray text means the document was cut short
	// or two documents were glued together.
	for {
		off := d.InputOffset()
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return root, nil
			}
			return nil, fmt.Errorf("goexml: line %d: %v", lt.at(off), err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return nil, fmt.Errorf("goexml: line %d: unexpected <%s> after root element", lt.at(off), t.Name.Local)
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return nil, fmt.Errorf("goexml: line %d: text outside root element", lt.at(off))
			}
		case xml.Comment, xml.ProcInst, xml.Directive:
			continue
		default:
			return nil, fmt.Errorf("goexml: line %d: unexpected token", lt.at(off))
		}
	}
}

// parseElement reads one element and its contents into a Node tree.
func parseElement(lt *lineTracker, d *xml.Decoder, start xml.StartElement, off int64) (*Node, error) {
	// Reject namespaced elements, whether the prefix is declared or not:
	// an undeclared prefix would otherwise parse as a plain local name
	// and slip an alien document through validation.
	if start.Name.Space != "" {
		return nil, fmt.Errorf("goexml: line %d: namespaces are not supported", lt.at(off))
	}
	n := &Node{Type: start.Name.Local, Attrs: map[string]string{}, Line: lt.at(off)}
	for _, a := range start.Attr {
		if a.Name.Space != "" {
			return nil, nodeErr(n, "namespaces are not supported")
		}
		if _, dup := n.Attrs[a.Name.Local]; dup {
			return nil, nodeErr(n, "duplicate attribute %q", a.Name.Local)
		}
		n.Attrs[a.Name.Local] = a.Value
	}
	var body strings.Builder
	for {
		coff := d.InputOffset()
		tok, err := d.Token()
		if err != nil {
			return nil, nodeErr(n, "inside: %v", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, err := parseElement(lt, d, t, coff)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, child)
		case xml.EndElement:
			if t.Name.Local != start.Name.Local {
				return nil, fmt.Errorf("goexml: line %d: mismatched </%s>", lt.at(coff), t.Name.Local)
			}
			n.Text = strings.TrimSpace(body.String())
			return n, nil
		case xml.CharData:
			body.WriteString(string(t))
		case xml.Comment:
			continue
		default:
			continue
		}
	}
}

// lineTracker converts byte offsets into 1-based line numbers without
// rescanning the document from byte 0 on every element. Decoder offsets
// move forward, so each query only scans the bytes since the last one.
type lineTracker struct {
	data []byte
	off  int64
	line int
}

func newLineTracker(data []byte) *lineTracker {
	return &lineTracker{data: data, line: 1}
}

// at returns the line containing the byte at off.
func (lt *lineTracker) at(off int64) int {
	if off < 0 {
		off = 0
	} else if off > int64(len(lt.data)) {
		off = int64(len(lt.data))
	}
	if off < lt.off {
		// Out-of-order query: rescan from the start to stay correct.
		return bytes.Count(lt.data[:off], []byte{'\n'}) + 1
	}
	lt.line += bytes.Count(lt.data[lt.off:off], []byte{'\n'})
	lt.off = off
	return lt.line
}

// nodeErr formats a node-scoped error, naming the element's id when it
// has one, per the coding rules: line, tag, and id if present. The
// format string must not repeat the "goexml: line" or tag prefix.
func nodeErr(n *Node, format string, args ...any) error {
	if id := n.Attr("id"); id != "" {
		return fmt.Errorf("goexml: line %d: <%s id=%q> "+format, append([]any{n.Line, n.Type, id}, args...)...)
	}
	return fmt.Errorf("goexml: line %d: <%s> "+format, append([]any{n.Line, n.Type}, args...)...)
}

// validate checks the whole tree against the 0.1.0 schema.
func validate(root *Node) error {
	if root.Type != "goexml" {
		return fmt.Errorf("goexml: line %d: root must be <goexml>", root.Line)
	}
	if v := root.Attr("version"); v != Version {
		return nodeErr(root, "unsupported version %q, want %q", v, Version)
	}
	ids := map[string]int{}
	if err := checkAttrs(root, ids); err != nil {
		return err
	}
	if len(root.Children) != 1 || root.Children[0].Type != "window" {
		return nodeErr(root, "must hold exactly one <window>")
	}
	return checkWindow(root.Children[0], ids)
}

func checkWindow(n *Node, ids map[string]int) error {
	if err := checkAttrs(n, ids); err != nil {
		return err
	}
	if err := checkValues(n); err != nil {
		return err
	}
	if len(n.Children) == 0 {
		return nodeErr(n, "needs content")
	}
	if len(n.Children) > 1 {
		// Multiple children would stack in an implicit, unconfigured
		// box; the author makes that layout explicit instead.
		return nodeErr(n, "needs exactly one child; wrap multiple children in a <vbox>")
	}
	for _, c := range n.Children {
		if !widgets[c.Type] {
			return nodeErr(c, "is not allowed inside <window>")
		}
		if err := checkWidget(c, "window", ids); err != nil {
			return err
		}
	}
	return nil
}

// checkWidget validates one widget element in the context of its parent.
func checkWidget(n *Node, parent string, ids map[string]int) error {
	if err := checkAttrs(n, ids); err != nil {
		return err
	}
	switch n.Type {
	case "vbox", "hbox":
		if mode, ok := n.Attrs["column"]; ok && mode != "grid" {
			return nodeErr(n, "unknown column mode %q, want grid on vbox", mode)
		}
		min, max := 0, 0
		if s, ok := n.Attrs["min"]; ok {
			if n.Attr("column") != "grid" {
				return nodeErr(n, "min needs column=\"grid\"")
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 {
				return nodeErr(n, "min must be an integer >= 1, got %q", s)
			}
			min = v
		}
		if s, ok := n.Attrs["max"]; ok {
			if n.Attr("column") != "grid" {
				return nodeErr(n, "max needs column=\"grid\"")
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 {
				return nodeErr(n, "max must be an integer >= 1, got %q", s)
			}
			max = v
		}
		if n.Attr("min") != "" && n.Attr("max") != "" && min > max {
			return nodeErr(n, "min %d exceeds max %d", min, max)
		}
		if len(n.Children) == 0 {
			return nodeErr(n, "needs at least one child")
		}
		for _, c := range n.Children {
			if !widgets[c.Type] {
				return nodeErr(c, "is not allowed inside <%s>", n.Type)
			}
			if err := checkWidget(c, n.Type, ids); err != nil {
				return err
			}
		}
	case "border":
		if len(n.Children) == 0 {
			return nodeErr(n, "needs at least one region")
		}
		seen := map[string]bool{}
		for _, c := range n.Children {
			if !regions[c.Type] {
				return nodeErr(c, "is not a border region")
			}
			if seen[c.Type] {
				return nodeErr(c, "duplicate <%s> region", c.Type)
			}
			seen[c.Type] = true
			if err := checkAttrs(c, ids); err != nil {
				return err
			}
			if len(c.Children) > 1 {
				// Multiple region children would stack in an implicit,
				// unconfigured box (see checkWindow).
				return nodeErr(c, "needs at most one child; wrap multiple children in a <vbox>")
			}
			for _, g := range c.Children {
				if !widgets[g.Type] {
					return nodeErr(g, "is not allowed inside <%s>", c.Type)
				}
				if err := checkWidget(g, c.Type, ids); err != nil {
					return err
				}
			}
		}
	case "label":
		if _, ok := n.Attrs["text"]; !ok {
			return nodeErr(n, "needs a text attribute")
		}
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
	case "button":
		if _, ok := n.Attrs["text"]; !ok {
			return nodeErr(n, "needs a text attribute")
		}
		if _, ok := n.Attrs["onClick"]; !ok {
			return nodeErr(n, "needs an onClick attribute")
		}
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
	case "text":
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
		v := strings.TrimSpace(n.Attr("v"))
		switch {
		case v != "" && n.Text != "":
			return nodeErr(n, "takes v attribute or inner text, not both")
		case v == "" && n.Text == "":
			return nodeErr(n, "needs text: v attribute or inner text")
		}
	case "entry":
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
	case "rect":
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
		if _, _, _, err := parseHexColor(n, n.Attr("color")); err != nil {
			return err
		}
	case "select":
		if _, ok := n.Attrs["onChange"]; !ok {
			return nodeErr(n, "needs an onChange attribute")
		}
		if len(n.Children) == 0 {
			return nodeErr(n, "needs at least one <option>")
		}
		picked := 0
		texts := map[string]int{}
		for _, o := range n.Children {
			if o.Type != "option" {
				return nodeErr(o, "is not allowed inside <select>")
			}
			if err := checkAttrs(o, ids); err != nil {
				return err
			}
			if len(o.Children) != 0 {
				return nodeErr(o, "cannot have children")
			}
			if o.Attr("value") == "" {
				return nodeErr(o, "needs a value attribute")
			}
			if o.Attr("text") == "" {
				return nodeErr(o, "needs a text attribute")
			}
			if prev, dup := texts[o.Attr("text")]; dup {
				return nodeErr(o, "duplicate option text %q, first seen on line %d", o.Attr("text"), prev)
			}
			texts[o.Attr("text")] = o.Line
			if sel, ok := o.Attrs["selected"]; ok {
				if sel != "" {
					return nodeErr(o, "selected takes no value, got %q", sel)
				}
				picked++
			}
		}
		if picked > 1 {
			return nodeErr(n, "allows only one selected <option>")
		}
	case "table":
		if len(n.Children) == 0 {
			return nodeErr(n, "needs at least one <row>")
		}
		cols := -1
		for _, r := range n.Children {
			if r.Type != "row" {
				return nodeErr(r, "is not allowed inside <table>")
			}
			if err := checkAttrs(r, ids); err != nil {
				return err
			}
			if len(r.Children) == 0 {
				return nodeErr(r, "needs at least one <cell>")
			}
			for _, c := range r.Children {
				if c.Type != "cell" {
					return nodeErr(c, "is not allowed inside <row>")
				}
				if err := checkAttrs(c, ids); err != nil {
					return err
				}
				if len(c.Children) != 0 {
					return nodeErr(c, "cannot have children")
				}
				attr := strings.TrimSpace(c.Attr("text"))
				switch {
				case attr != "" && c.Text != "":
					return nodeErr(c, "takes text attribute or inner text, not both")
				case attr == "" && c.Text == "":
					return nodeErr(c, "needs text: text attribute or inner text")
				}
			}
			if cols < 0 {
				cols = len(r.Children)
			} else if len(r.Children) != cols {
				return nodeErr(r, "has %d cells, want %d", len(r.Children), cols)
			}
		}
	default:
		return nodeErr(n, "unknown element inside <%s>", parent)
	}
	return checkValues(n)
}

// checkAttrs rejects unknown attributes and tracks duplicate ids.
// It also rejects inner text everywhere except cell and text elements,
// which is checked here because every node passes through this function.
func checkAttrs(n *Node, ids map[string]int) error {
	if _, known := tagAttrs[n.Type]; !known {
		return nodeErr(n, "unknown element")
	}
	if n.Type != "cell" && n.Type != "text" && n.Text != "" {
		return nodeErr(n, "text is not allowed")
	}
	allowed := allowedAttrs(n.Type)
	var unknown []string
	for name := range n.Attrs {
		if !allowed[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		// Map iteration order is random; sorting makes the message
		// deterministic and names every offending attribute.
		sort.Strings(unknown)
		quoted := make([]string, len(unknown))
		for i, a := range unknown {
			quoted[i] = fmt.Sprintf("%q", a)
		}
		return nodeErr(n, "has no attribute %s", strings.Join(quoted, ", "))
	}
	if id := n.Attr("id"); id != "" {
		if !idPattern.MatchString(id) {
			return nodeErr(n, "id must use letters, digits or _, got %q", id)
		}
		if prev, dup := ids[id]; dup {
			return nodeErr(n, "duplicate id, first seen on line %d", prev)
		}
		ids[id] = n.Line
	} else if idRequired[n.Type] {
		return nodeErr(n, "needs an id attribute")
	}
	return nil
}

// checkValues validates attribute value formats.
func checkValues(n *Node) error {
	for _, key := range []string{"visible", "disabled"} {
		if v := n.Attr(key); v != "" && v != "true" && v != "false" {
			return nodeErr(n, "%s must be true or false, got %q", key, v)
		}
	}
	for _, key := range []string{"padding", "spacing"} {
		if v := n.Attr(key); v != "" {
			if i, err := strconv.Atoi(v); err != nil || i < 0 {
				return nodeErr(n, "%s must be a non-negative integer, got %q", key, v)
			}
		}
	}
	if v := n.Attr("align"); v != "" && v != "start" && v != "center" && v != "end" {
		return nodeErr(n, "align must be start, center or end, got %q", v)
	}
	switch n.Attr("wrap") {
	case "", "off", "word", "break":
	default:
		return nodeErr(n, "wrap must be word, break or off, got %q", n.Attr("wrap"))
	}
	switch n.Attr("justify") {
	case "", "start", "center", "end", "space-between", "space-around", "space-evenly":
	default:
		return nodeErr(n, "justify must be start, center, end, space-between, space-around or space-evenly, got %q", n.Attr("justify"))
	}
	st, err := styles.ParseStyle(n.Attr("style"))
	if err != nil {
		return nodeErr(n, "invalid style: %v", err)
	}
	// tableLayout never shrinks below the measured content size, so an
	// exact height (or a fill height smaller than the content) would
	// overflow the slot and overlap siblings. A table always sizes to
	// its content.
	if n.Type == "table" && (st.HFull || st.HasFixedH()) {
		return nodeErr(n, "style h-* is not supported on a table; a table sizes to its content")
	}
	if n.Type == "window" {
		if t := n.Attr("title"); t == "" {
			return nodeErr(n, "needs a title attribute")
		}
		if _, _, err := parseSize(n, n.Attr("size")); err != nil {
			return err
		}
	}
	return nil
}

// parseHexColor parses a "#rrggbb" color.
func parseHexColor(n *Node, s string) (r, g, b uint8, err error) {
	bad := func() (uint8, uint8, uint8, error) {
		return 0, 0, 0, nodeErr(n, "color must look like #2e2e2e, got %q", s)
	}
	if len(s) != 7 || s[0] != '#' {
		return bad()
	}
	var v [3]uint8
	for i := 0; i < 3; i++ {
		n, perr := strconv.ParseUint(s[1+2*i:3+2*i], 16, 8)
		if perr != nil {
			return bad()
		}
		v[i] = uint8(n)
	}
	return v[0], v[1], v[2], nil
}

// parseSize parses a "WxH" size such as "800x600".
func parseSize(n *Node, s string) (w, h int, err error) {
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, nodeErr(n, "size must look like 800x600, got %q", s)
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, nodeErr(n, "size must look like 800x600, got %q", s)
	}
	return w, h, nil
}
