package goexml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/five-zombie/goexmlfyne/styles"
)

// Decode parses a goexml 0.1.0 document and validates it against the
// schema. It returns the root node. Every error carries the 1-based
// source line of the offending element.
func Decode(data []byte) (*Node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		off := d.InputOffset()
		tok, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("goexml: line %d: %v", lineAt(data, off), err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			root, err := parseElement(d, data, t, off)
			if err != nil {
				return nil, err
			}
			if err := validate(root); err != nil {
				return nil, err
			}
			return root, nil
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return nil, fmt.Errorf("goexml: line %d: text outside root element", lineAt(data, off))
			}
		case xml.Comment, xml.ProcInst, xml.Directive:
			continue
		default:
			return nil, fmt.Errorf("goexml: line %d: unexpected token", lineAt(data, off))
		}
	}
}

// parseElement reads one element and its children into a Node tree.
func parseElement(d *xml.Decoder, data []byte, start xml.StartElement, off int64) (*Node, error) {
	n := &Node{Type: start.Name.Local, Attrs: map[string]string{}, Line: lineAt(data, off)}
	for _, a := range start.Attr {
		if a.Name.Space != "" {
			return nil, fmt.Errorf("goexml: line %d: namespaces are not supported", n.Line)
		}
		n.Attrs[a.Name.Local] = a.Value
	}
	var body strings.Builder
	for {
		coff := d.InputOffset()
		tok, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("goexml: line %d: inside <%s>: %v", n.Line, n.Type, err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, err := parseElement(d, data, t, coff)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, child)
		case xml.EndElement:
			if t.Name.Local != start.Name.Local {
				return nil, fmt.Errorf("goexml: line %d: mismatched </%s>", lineAt(data, coff), t.Name.Local)
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

// lineAt converts a byte offset into a 1-based line number.
func lineAt(data []byte, off int64) int {
	if off < 0 {
		off = 0
	}
	if off > int64(len(data)) {
		off = int64(len(data))
	}
	return bytes.Count(data[:off], []byte{'\n'}) + 1
}

// validate checks the whole tree against the 0.1.0 schema.
func validate(root *Node) error {
	if root.Type != "goexml" {
		return fmt.Errorf("goexml: line %d: root must be <goexml>", root.Line)
	}
	if v := root.Attr("version"); v != Version {
		return fmt.Errorf("goexml: line %d: unsupported version %q, want %q", root.Line, v, Version)
	}
	ids := map[string]int{}
	if err := checkAttrs(root, ids); err != nil {
		return err
	}
	if len(root.Children) != 1 || root.Children[0].Type != "window" {
		return fmt.Errorf("goexml: line %d: <goexml> must hold exactly one <window>", root.Line)
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
		return fmt.Errorf("goexml: line %d: <window> needs content", n.Line)
	}
	for _, c := range n.Children {
		if !widgets[c.Type] {
			return fmt.Errorf("goexml: line %d: <%s> is not allowed inside <window>", c.Line, c.Type)
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
		if mode, ok := n.Attrs["column"]; ok && (n.Type != "vbox" || mode != "grid") {
			return fmt.Errorf("goexml: line %d: unknown column mode %q, want grid on vbox", n.Line, mode)
		}
		min, max := 0, 0
		if s, ok := n.Attrs["min"]; ok {
			if n.Attr("column") != "grid" {
				return fmt.Errorf("goexml: line %d: min needs column=\"grid\"", n.Line)
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 {
				return fmt.Errorf("goexml: line %d: min must be an integer >= 1, got %q", n.Line, s)
			}
			min = v
		}
		if s, ok := n.Attrs["max"]; ok {
			if n.Attr("column") != "grid" {
				return fmt.Errorf("goexml: line %d: max needs column=\"grid\"", n.Line)
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 {
				return fmt.Errorf("goexml: line %d: max must be an integer >= 1, got %q", n.Line, s)
			}
			max = v
		}
		if n.Attr("min") != "" && n.Attr("max") != "" && min > max {
			return fmt.Errorf("goexml: line %d: min %d exceeds max %d", n.Line, min, max)
		}
		if len(n.Children) == 0 {
			return fmt.Errorf("goexml: line %d: <%s> needs at least one child", n.Line, n.Type)
		}
		for _, c := range n.Children {
			if !widgets[c.Type] {
				return fmt.Errorf("goexml: line %d: <%s> is not allowed inside <%s>", c.Line, c.Type, n.Type)
			}
			if err := checkWidget(c, n.Type, ids); err != nil {
				return err
			}
		}
	case "border":
		if len(n.Children) == 0 {
			return fmt.Errorf("goexml: line %d: <border> needs at least one region", n.Line)
		}
		seen := map[string]bool{}
		for _, c := range n.Children {
			if !regions[c.Type] {
				return fmt.Errorf("goexml: line %d: <%s> is not a border region", c.Line, c.Type)
			}
			if seen[c.Type] {
				return fmt.Errorf("goexml: line %d: duplicate <%s> region", c.Line, c.Type)
			}
			seen[c.Type] = true
			if err := checkAttrs(c, ids); err != nil {
				return err
			}
			for _, g := range c.Children {
				if !widgets[g.Type] {
					return fmt.Errorf("goexml: line %d: <%s> is not allowed inside <%s>", g.Line, g.Type, c.Type)
				}
				if err := checkWidget(g, c.Type, ids); err != nil {
					return err
				}
			}
		}
	case "label":
		if _, ok := n.Attrs["text"]; !ok {
			return fmt.Errorf("goexml: line %d: <label> needs a text attribute", n.Line)
		}
		if len(n.Children) != 0 {
			return fmt.Errorf("goexml: line %d: <label> cannot have children", n.Line)
		}
	case "button":
		if _, ok := n.Attrs["text"]; !ok {
			return fmt.Errorf("goexml: line %d: <button> needs a text attribute", n.Line)
		}
		if _, ok := n.Attrs["onClick"]; !ok {
			return fmt.Errorf("goexml: line %d: <button> needs an onClick attribute", n.Line)
		}
		if len(n.Children) != 0 {
			return fmt.Errorf("goexml: line %d: <button> cannot have children", n.Line)
		}
	case "text":
		if len(n.Children) != 0 {
			return fmt.Errorf("goexml: line %d: <text> cannot have children", n.Line)
		}
		v := strings.TrimSpace(n.Attr("v"))
		switch {
		case v != "" && n.Text != "":
			return fmt.Errorf("goexml: line %d: <text> takes v attribute or inner text, not both", n.Line)
		case v == "" && n.Text == "":
			return fmt.Errorf("goexml: line %d: <text> needs text: v attribute or inner text", n.Line)
		}
	case "entry":
		if len(n.Children) != 0 {
			return fmt.Errorf("goexml: line %d: <entry> cannot have children", n.Line)
		}
	case "rect":
		if len(n.Children) != 0 {
			return fmt.Errorf("goexml: line %d: <rect> cannot have children", n.Line)
		}
		if _, _, _, err := parseHexColor(n.Attr("color"), n.Line); err != nil {
			return err
		}
	case "select":
		if _, ok := n.Attrs["onChange"]; !ok {
			return fmt.Errorf("goexml: line %d: <select> needs an onChange attribute", n.Line)
		}
		if len(n.Children) == 0 {
			return fmt.Errorf("goexml: line %d: <select> needs at least one <option>", n.Line)
		}
		picked := 0
		for _, o := range n.Children {
			if o.Type != "option" {
				return fmt.Errorf("goexml: line %d: <%s> is not allowed inside <select>", o.Line, o.Type)
			}
			if err := checkAttrs(o, ids); err != nil {
				return err
			}
			if len(o.Children) != 0 {
				return fmt.Errorf("goexml: line %d: <option> cannot have children", o.Line)
			}
			if o.Attr("value") == "" {
				return fmt.Errorf("goexml: line %d: <option> needs a value attribute", o.Line)
			}
			if o.Attr("text") == "" {
				return fmt.Errorf("goexml: line %d: <option> needs a text attribute", o.Line)
			}
			if sel, ok := o.Attrs["selected"]; ok {
				if sel != "true" {
					return fmt.Errorf("goexml: line %d: <option> selected must be true, got %q", o.Line, sel)
				}
				picked++
			}
		}
		if picked > 1 {
			return fmt.Errorf("goexml: line %d: <select> allows only one selected <option>", n.Line)
		}
	case "table":
		if len(n.Children) == 0 {
			return fmt.Errorf("goexml: line %d: <table> needs at least one <row>", n.Line)
		}
		cols := -1
		for _, r := range n.Children {
			if r.Type != "row" {
				return fmt.Errorf("goexml: line %d: <%s> is not allowed inside <table>", r.Line, r.Type)
			}
			if err := checkAttrs(r, ids); err != nil {
				return err
			}
			if len(r.Children) == 0 {
				return fmt.Errorf("goexml: line %d: <row> needs at least one <cell>", r.Line)
			}
			for _, c := range r.Children {
				if c.Type != "cell" {
					return fmt.Errorf("goexml: line %d: <%s> is not allowed inside <row>", c.Line, c.Type)
				}
				if err := checkAttrs(c, ids); err != nil {
					return err
				}
				if len(c.Children) != 0 {
					return fmt.Errorf("goexml: line %d: <cell> cannot have children", c.Line)
				}
				attr := strings.TrimSpace(c.Attr("text"))
				switch {
				case attr != "" && c.Text != "":
					return fmt.Errorf("goexml: line %d: <cell> takes text attribute or inner text, not both", c.Line)
				case attr == "" && c.Text == "":
					return fmt.Errorf("goexml: line %d: <cell> needs text: text attribute or inner text", c.Line)
				}
			}
			if cols < 0 {
				cols = len(r.Children)
			} else if len(r.Children) != cols {
				return fmt.Errorf("goexml: line %d: <row> has %d cells, want %d", r.Line, len(r.Children), cols)
			}
		}
	default:
		return fmt.Errorf("goexml: line %d: unknown element <%s> inside <%s>", n.Line, n.Type, parent)
	}
	return checkValues(n)
}

// checkAttrs rejects unknown attributes and tracks duplicate ids.
// It also rejects inner text everywhere except cell and text elements,
// which is checked here because every node passes through this function.
func checkAttrs(n *Node, ids map[string]int) error {
	if _, known := tagAttrs[n.Type]; !known {
		return fmt.Errorf("goexml: line %d: unknown element <%s>", n.Line, n.Type)
	}
	if n.Type != "cell" && n.Type != "text" && n.Text != "" {
		return fmt.Errorf("goexml: line %d: text is not allowed inside <%s>", n.Line, n.Type)
	}
	allowed := allowedAttrs(n.Type)
	for name := range n.Attrs {
		if !allowed[name] {
			return fmt.Errorf("goexml: line %d: <%s> has no attribute %q", n.Line, n.Type, name)
		}
	}
	if id := n.Attr("id"); id != "" {
		if prev, dup := ids[id]; dup {
			return fmt.Errorf("goexml: line %d: duplicate id %q, first seen on line %d", n.Line, id, prev)
		}
		ids[id] = n.Line
	} else if idRequired[n.Type] {
		return fmt.Errorf("goexml: line %d: <%s> needs an id attribute", n.Line, n.Type)
	}
	return nil
}

// checkValues validates attribute value formats.
func checkValues(n *Node) error {
	for _, key := range []string{"visible", "disabled", "expand"} {
		if v := n.Attr(key); v != "" && v != "true" && v != "false" {
			return fmt.Errorf("goexml: line %d: <%s> %s must be true or false, got %q", n.Line, n.Type, key, v)
		}
	}
	for _, key := range []string{"padding", "spacing"} {
		if v := n.Attr(key); v != "" {
			if i, err := strconv.Atoi(v); err != nil || i < 0 {
				return fmt.Errorf("goexml: line %d: <%s> %s must be a non-negative integer, got %q", n.Line, n.Type, key, v)
			}
		}
	}
	if v := n.Attr("align"); v != "" && v != "start" && v != "center" && v != "end" {
		return fmt.Errorf("goexml: line %d: <%s> align must be start, center or end, got %q", n.Line, n.Type, v)
	}
	switch n.Attr("wrap") {
	case "", "off", "word", "break":
	default:
		return fmt.Errorf("goexml: line %d: <%s> wrap must be word or break, got %q", n.Line, n.Type, n.Attr("wrap"))
	}
	switch n.Attr("justify") {
	case "", "start", "center", "end", "space-between", "space-around", "space-evenly":
	default:
		return fmt.Errorf("goexml: line %d: <%s> justify must be start, center, end, space-between, space-around or space-evenly, got %q", n.Line, n.Type, n.Attr("justify"))
	}
	if _, err := styles.ParseStyle(n.Attr("style")); err != nil {
		return fmt.Errorf("goexml: line %d: <%s> has %v", n.Line, n.Type, err)
	}
	if n.Type == "window" {
		if t := n.Attr("title"); t == "" {
			return fmt.Errorf("goexml: line %d: <window> needs a title attribute", n.Line)
		}
		if _, _, err := parseSize(n.Attr("size"), n.Line); err != nil {
			return err
		}
	}
	return nil
}

// parseHexColor parses a "#rrggbb" color.
func parseHexColor(s string, line int) (r, g, b uint8, err error) {
	bad := func() (uint8, uint8, uint8, error) {
		return 0, 0, 0, fmt.Errorf("goexml: line %d: color must look like #2e2e2e, got %q", line, s)
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
func parseSize(s string, line int) (w, h int, err error) {
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("goexml: line %d: size must look like 800x600, got %q", line, s)
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("goexml: line %d: size must look like 800x600, got %q", line, s)
	}
	return w, h, nil
}
