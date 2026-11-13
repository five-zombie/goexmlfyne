package goexml

import (
	"bytes"
	"encoding/xml"
	"errors"
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
	d, lt, start, off, err := openEnvelope(patchFlags(data))
	if err != nil {
		return nil, err
	}
	root, err := parseElement(lt, d, start, off)
	if err != nil {
		return nil, err
	}
	if err := validate(root); err != nil {
		return nil, err
	}
	if err := finishTrailing(lt, d, root); err != nil {
		return nil, err
	}
	return root, nil
}

// DecodeItem parses a goexml-item fragment: the injectable envelope
// that holds exactly one child widget and has no window. The same
// schema rules apply as for a document. The fragment's ids join the
// document's global id space, which Build and BuildItem enforce.
func DecodeItem(data []byte) (*Node, error) {
	d, lt, start, off, err := openEnvelope(patchFlags(data))
	if err != nil {
		return nil, err
	}
	root, err := parseElement(lt, d, start, off)
	if err != nil {
		return nil, err
	}
	if root.Type != "goexml-item" {
		return nil, fmt.Errorf("goexml: line %d: fragment root must be <goexml-item>, got <%s>", root.Line, root.Type)
	}
	if v := root.Attr("version"); v != Version {
		return nil, nodeErr(root, "unsupported version %q, want %q", v, Version)
	}
	ids := map[string]int{}
	if err := checkAttrs(root, ids); err != nil {
		return nil, err
	}
	if len(root.Children) != 1 {
		return nil, nodeErr(root, "must hold exactly one child widget; wrap multiple children in a <vbox>")
	}
	child := root.Children[0]
	if !widgets[child.Type] {
		return nil, nodeErr(child, "is not allowed inside <goexml-item>")
	}
	if err := checkWidget(child, "goexml-item", ids, false); err != nil {
		return nil, err
	}
	if err := finishTrailing(lt, d, root); err != nil {
		return nil, err
	}
	return root, nil
}

// openEnvelope strips the BOM, prepares the decoder, and seeks the
// first element past the prolog. Both envelopes share it: the document
// (goexml) and the fragment (goexml-item).
func openEnvelope(data []byte) (d *xml.Decoder, lt *lineTracker, start xml.StartElement, off int64, err error) {
	// A UTF-8 BOM is common from Windows editors; strip it instead of
	// misreporting it as text outside the root element.
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	d = xml.NewDecoder(bytes.NewReader(data))
	lt = newLineTracker(data)
	for {
		off := d.InputOffset()
		tok, terr := d.Token()
		if terr != nil {
			if terr == io.EOF {
				return d, lt, start, off, fmt.Errorf("goexml: line %d: document has no root element", lt.at(off))
			}
			return d, lt, start, off, syntaxErr(terr, lt.at(off))
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return d, lt, t, off, nil
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return d, lt, start, off, fmt.Errorf("goexml: line %d: text outside root element", lt.at(off))
			}
		case xml.Comment, xml.ProcInst, xml.Directive:
			continue
		default:
			return d, lt, start, off, fmt.Errorf("goexml: line %d: unexpected token", lt.at(off))
		}
	}
}

// finishTrailing rejects anything but whitespace and comments after the
// root: a second element or stray text means the document was cut short
// or two documents were glued together.
func finishTrailing(lt *lineTracker, d *xml.Decoder, root *Node) error {
	for {
		off := d.InputOffset()
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return syntaxErr(err, lt.at(off))
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return fmt.Errorf("goexml: line %d: unexpected <%s> after root element", lt.at(off), t.Name.Local)
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				return fmt.Errorf("goexml: line %d: text outside root element", lt.at(off))
			}
		case xml.Comment, xml.ProcInst, xml.Directive:
			continue
		default:
			return fmt.Errorf("goexml: line %d: unexpected token", lt.at(off))
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
			return nil, syntaxErr(err, lt.at(off))
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
		case xml.Directive:
			// Directives are only legal in the prolog; inside an
			// element they are almost certainly misplaced.
			return nil, nodeErr(n, "directives are not allowed inside an element")
		case xml.ProcInst:
			// Processing instructions are legal XML content, but the
			// goexml standard has no use for them, so they are
			// rejected rather than silently dropped.
			return nil, nodeErr(n, "processing instructions are not allowed inside an element")
		default:
			return nil, nodeErr(n, "unexpected token")
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

// syntaxErr reports a decoder error with a single line number: XML
// syntax errors carry their own line, so that one is used as the only
// line number instead of pairing it with the tracked token position.
func syntaxErr(err error, line int) error {
	var se *xml.SyntaxError
	if errors.As(err, &se) {
		return fmt.Errorf("goexml: line %d: %s", se.Line, se.Msg)
	}
	return fmt.Errorf("goexml: line %d: %v", line, err)
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
		if err := checkWidget(c, "window", ids, false); err != nil {
			return err
		}
	}
	return nil
}

// checkWidget validates one widget element in the context of its parent.
// inContainer marks that an ancestor <container> encloses this subtree:
// slots are only legal there, and a nested container is rejected.
func checkWidget(n *Node, parent string, ids map[string]int, inContainer bool) error {
	if err := checkAttrs(n, ids); err != nil {
		return err
	}
	switch {
	case n.Type == "container" && inContainer:
		// One container is one injection unit; nesting would blur who
		// owns which slot. Siblings stay fine, and runtime composition
		// through injection is unaffected: the rule is per file.
		return nodeErr(n, "cannot be nested inside <container>")
	case n.Type == "slot" && !inContainer:
		// A slot outside a container has no owner to fill it — exactly
		// the "accepted then dead" pattern this parser rejects.
		return nodeErr(n, "is only allowed inside a <container>")
	}
	switch n.Type {
	case "container":
		if len(n.Children) != 1 {
			return nodeErr(n, "needs exactly one child; wrap multiple children in a <vbox>")
		}
		if got := countSlots(n); got != 1 {
			return nodeErr(n, "needs exactly one <slot> in its subtree, got %d", got)
		}
		return checkWidget(n.Children[0], "container", ids, true)
	case "slot":
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
		// A slot is a hole: the visual layer has nothing to paint on.
		// The style parser accepts the tokens generically, so reject
		// them here instead of letting them decode and render nothing.
		if st, err := styles.ParseStyle(n.Attr("style")); err == nil && st.Vis.Active() {
			return nodeErr(n, "renders no visual layer: drop bg-*, b-* and rounded from its style")
		}
	case "vbox-expand":
		// Items come from the runtime only: a child here would never
		// reach the screen and would silently rot. Layout attributes
		// mirror the static vbox — spacing, align, justify, and the
		// grid column block — because a committed item is laid out by
		// the same engine.
		if len(n.Children) != 0 {
			return nodeErr(n, "takes no children: fill it at runtime with Draft and Commit")
		}
		// The flag means true when bare (patchFlags rewrites it) or
		// when valued "true"; the older empty form and any other
		// value point the author at the accepted spellings.
		if s, ok := n.Attrs["scrollable"]; ok && s != "true" {
			return nodeErr(n, "scrollable must be the bare attribute or scrollable=\"true\", got %q", s)
		}
		if mode, ok := n.Attrs["column"]; ok && mode != "grid" {
			return nodeErr(n, "unknown column mode %q, want grid on vbox-expand", mode)
		}
		for _, key := range []string{"min", "max"} {
			if s, ok := n.Attrs[key]; ok {
				if n.Attr("column") != "grid" {
					return nodeErr(n, "%s needs column=\"grid\"", key)
				}
				v, err := strconv.Atoi(s)
				if err != nil || v < 1 {
					return nodeErr(n, "%s must be an integer >= 1, got %q", key, s)
				}
			}
		}
		if n.Attr("min") != "" && n.Attr("max") != "" {
			minV, _ := strconv.Atoi(n.Attr("min"))
			maxV, _ := strconv.Atoi(n.Attr("max"))
			if minV > maxV {
				return nodeErr(n, "min %d exceeds max %d", minV, maxV)
			}
		}
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
			if err := checkWidget(c, n.Type, ids, inContainer); err != nil {
				return err
			}
		}
	case "border":
		// The five-region border element was an invented design and is
		// removed from the schema; it reaches here only from hand-built
		// trees, since decode no longer knows the tag.
		return nodeErr(n, "was removed from the schema; use vbox/hbox")
	case "label":
		if _, ok := n.Attrs["text"]; !ok {
			return nodeErr(n, "needs a text attribute")
		}
		if err := checkPlaceholders(n, n.Attr("text")); err != nil {
			return err
		}
		if err := checkTemplateID(n, n.Attr("text")); err != nil {
			return err
		}
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
	case "button":
		if _, ok := n.Attrs["text"]; !ok {
			return nodeErr(n, "needs a text attribute")
		}
		if err := checkPlaceholders(n, n.Attr("text")); err != nil {
			return err
		}
		if err := checkTemplateID(n, n.Attr("text")); err != nil {
			return err
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
		if err := checkPlaceholders(n, v); err != nil {
			return err
		}
		if err := checkTemplateID(n, v); err != nil {
			return err
		}
	case "entry":
		if err := checkPlaceholders(n, n.Attr("text")); err != nil {
			return err
		}
		if err := checkTemplateID(n, n.Attr("text")); err != nil {
			return err
		}
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
	case "image":
		if len(n.Children) != 0 {
			return nodeErr(n, "cannot have children")
		}
		if _, hasSrc := n.Attrs["src"]; !hasSrc && n.Attr("id") == "" {
			return nodeErr(n, "needs an id attribute: a slot image must be addressable")
		}
		if src, ok := n.Attrs["src"]; ok {
			if src == "" {
				return nodeErr(n, "bad src %q, want a resource key or no src attribute at all", src)
			}
			// src names a resource the runtime registers, never a
			// file or URL: a fixed key alphabet keeps a stray path or
			// template out. Absent src means a runtime-filled slot.
			for _, r := range src {
				if !('A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' || '0' <= r && r <= '9' ||
					r == '.' || r == '-' || r == '/' || r == '_') {
					return nodeErr(n, "bad src %q, want a resource key of letters, digits, dot, dash, slash or underscore", src)
				}
			}
		}
		switch n.Attr("fit") {
		case "", "contain", "cover", "fill":
		default:
			return nodeErr(n, "fit must be contain, cover or fill, got %q", n.Attr("fit"))
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
			if err := checkTemplateID(n, o.Attr("text")); err != nil {
				return err
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
			if err := checkPlaceholders(o, o.Attr("text")); err != nil {
				return err
			}
			if prev, dup := texts[o.Attr("text")]; dup {
				return nodeErr(o, "duplicate option text %q, first seen on line %d", o.Attr("text"), prev)
			}
			texts[o.Attr("text")] = o.Line
			if sel, ok := o.Attrs["selected"]; ok {
				if sel != "true" {
					return nodeErr(o, "selected must be the bare attribute or selected=\"true\", got %q", sel)
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

// countSlots counts slot elements in a container subtree. The
// anti-nesting rule guarantees they all belong to this container.
func countSlots(n *Node) int {
	c := 0
	if n.Type == "slot" {
		c++
	}
	for _, ch := range n.Children {
		c += countSlots(ch)
	}
	return c
}

// checkPlaceholders validates {{token}} placeholders in a dynamic text
// attribute: every {{ opens a token of letters, digits and underscore
// closed by }}. An unclosed or malformed group is rejected here, not
// silently rendered as literal text.
func checkPlaceholders(n *Node, s string) error {
	for i := 0; i < len(s); {
		open := strings.Index(s[i:], "{{")
		if open < 0 {
			return nil
		}
		i += open
		close := strings.Index(s[i+2:], "}}")
		if close < 0 {
			return nodeErr(n, "unclosed placeholder %q, want {{token}}", s[i:])
		}
		tok := s[i+2 : i+2+close]
		if !idPattern.MatchString(tok) {
			return nodeErr(n, "invalid placeholder {{%s}}, want letters, digits or _", tok)
		}
		i += close + 4
	}
	return nil
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
			// The padding attribute is gone from vbox/hbox: point old
			// documents at the style utilities instead of a generic
			// unknown-attribute error.
			if name == "padding" && (n.Type == "vbox" || n.Type == "hbox") {
				return nodeErr(n, "padding moved to style: use the p-N utilities, e.g. p-2 for 8px")
			}
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

// checkTemplateID enforces the conditional id rule: a {{placeholder}}
// template only renders through the runtime's format setters, which
// address widgets by id, so a templated element without id would be a
// dead token. vals are the node's template-carrying attributes;
// n.Text joins the check because <text> may hold its body inline.
func checkTemplateID(n *Node, vals ...string) error {
	if n.Attr("id") != "" {
		return nil
	}
	for _, v := range vals {
		if strings.Contains(v, "{{") {
			return nodeErr(n, "needs an id attribute: a {{placeholder}} template renders through the runtime's setters")
		}
	}
	if strings.Contains(n.Text, "{{") {
		return nodeErr(n, "needs an id attribute: a {{placeholder}} template renders through the runtime's setters")
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
	for _, key := range []string{"spacing"} {
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
