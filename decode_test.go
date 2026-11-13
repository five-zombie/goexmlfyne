package goexml

import (
	"strings"
	"testing"
)

const validDoc = `<goexml version="0.1.0">
  <window title="Form" size="800x600">
    <vbox id="main" padding="8" spacing="4" align="start">
      <label id="lblTitle" text="Enter name" />
      <entry id="inpName" text="" placeholder="Name" style="w-full" />
      <hbox id="actions" spacing="4" align="end">
        <button id="btnCancel" text="Cancel" onClick="OnCancel" />
        <button id="btnSave" text="Save" onClick="OnSave" />
      </hbox>
    </vbox>
  </window>
</goexml>`

func TestDecodeValid(t *testing.T) {
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if root.Type != "goexml" {
		t.Fatalf("root type = %q, want goexml", root.Type)
	}
	win := root.Children[0]
	if win.Attr("title") != "Form" {
		t.Fatalf("title = %q, want Form", win.Attr("title"))
	}
	box := win.Children[0]
	if len(box.Children) != 3 {
		t.Fatalf("vbox children = %d, want 3", len(box.Children))
	}
	if box.Children[0].Line != 4 {
		t.Fatalf("label line = %d, want 4", box.Children[0].Line)
	}
}

func TestDecodeSelectTable(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><select id="s" onChange="OnPick">` +
		`<option value="a" text="A"/><option value="b" text="B, special" selected=""/></select>` +
		`<table id="t"><row><cell text="Name"/><cell text="Age"/></row>` +
		`<row><cell text="Ann"/><cell text="30"/></row></table>` +
		`</vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	box := root.Children[0].Children[0]
	if box.Children[0].Type != "select" || box.Children[1].Type != "table" {
		t.Fatalf("children = %q and %q, want select and table",
			box.Children[0].Type, box.Children[1].Type)
	}
}

func TestDecodeGrid(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="g" column="grid" min="1" max="5">` +
		`<label id="a" text="A"/><label id="b" text="B"/></vbox></window></goexml>`
	if _, err := Decode([]byte(doc)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
}

func TestDecodeTextForms(t *testing.T) {
	docs := map[string]string{
		"body":   `<goexml version="0.1.0"><window title="x" size="10x10"><vbox id="b"><text id="t">Hello world</text></vbox></window></goexml>`,
		"v attr": `<goexml version="0.1.0"><window title="x" size="10x10"><vbox id="b"><text id="t" v="Hello world"/></vbox></window></goexml>`,
		"cell":   `<goexml version="0.1.0"><window title="x" size="10x10"><table id="t"><row><cell>Ann</cell><cell>Designer</cell></row></table></window></goexml>`,
	}
	for name, doc := range docs {
		root, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("Decode %s returned error: %v", name, err)
		}
		_ = root
	}
	root, _ := Decode([]byte(docs["body"]))
	if got := root.Children[0].Children[0].Children[0].Text; got != "Hello world" {
		t.Fatalf("inner text = %q, want Hello world", got)
	}
	root, _ = Decode([]byte(docs["cell"]))
	row := root.Children[0].Children[0].Children[0]
	if got := row.Children[0].Text; got != "Ann" {
		t.Fatalf("cell inner text = %q, want Ann", got)
	}
}

func TestDecodeTrailingComment(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x"/></window></goexml><!-- done -->  `
	if _, err := Decode([]byte(doc)); err != nil {
		t.Fatalf("Decode returned error for trailing comment: %v", err)
	}
}

func TestDecodeBOM(t *testing.T) {
	doc := "\ufeff" + validDoc
	if _, err := Decode([]byte(doc)); err != nil {
		t.Fatalf("Decode returned error for BOM-prefixed document: %v", err)
	}
}

func TestDecodeEmpty(t *testing.T) {
	for _, data := range []string{"", "\n  ", "<!-- only a comment -->"} {
		_, err := Decode([]byte(data))
		if err == nil || !strings.Contains(err.Error(), "no root element") {
			t.Fatalf("Decode(%q) = %v, want no-root-element error", data, err)
		}
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want string
	}{
		{"bad version", `<goexml version="9.9"><window title="x" size="1x1"><label id="a" text="x"/></window></goexml>`, "unsupported version"},
		{"unknown tag", `<goexml version="0.1.0"><window title="x" size="1x1"><video id="a"/></window></goexml>`, "not allowed"},
		{"unknown attr", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x" color="red"/></window></goexml>`, "no attribute"},
		{"template without id", `<goexml version="0.1.0"><window title="x" size="1x1"><label text="{{name}}"/></window></goexml>`, "needs an id attribute"},
		{"dup id", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="a"><label id="a" text="x"/></vbox></window></goexml>`, "duplicate id"},
		{"two windows", `<goexml version="0.1.0"><window title="x" size="1x1"/><window title="y" size="1x1"/></goexml>`, "exactly one"},
		{"bad size", `<goexml version="0.1.0"><window title="x" size="big"><label id="a" text="x"/></window></goexml>`, "size must look like"},
		{"bad align", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="a" align="middle"><label id="b" text="x"/></vbox></window></goexml>`, "align must be"},
		{"text content", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x">hi</label></window></goexml>`, "text is not allowed"},
		{"button no handler attr", `<goexml version="0.1.0"><window title="x" size="1x1"><button id="a" text="x"/></window></goexml>`, "needs an onClick"},
		{"dup region", `<goexml version="0.1.0"><window title="x" size="1x1"><border id="b"><top><label id="a" text="x"/></top><top/></border></window></goexml>`, "duplicate <top>"},
		{"select no options", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s" onChange="OnPick"/></window></goexml>`, "at least one <option>"},
		{"select two selected", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s" onChange="OnPick"><option value="a" text="A" selected=""/><option value="b" text="B" selected=""/></select></window></goexml>`, "only one selected"},
		{"select valued selected", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s" onChange="OnPick"><option value="a" text="A" selected="true"/></select></window></goexml>`, "selected takes no value"},
		{"select option no value", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s" onChange="OnPick"><option text="A"/></select></window></goexml>`, "needs a value attribute"},
		{"select bad child", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s" onChange="OnPick"><label id="l" text="x"/></select></window></goexml>`, "not allowed inside <select>"},
		{"select no handler attr", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s"><option value="a" text="A"/></select></window></goexml>`, "needs an onChange"},
		{"select dup option text", `<goexml version="0.1.0"><window title="x" size="1x1"><select id="s" onChange="OnPick"><option value="a" text="A"/><option value="b" text="A"/></select></window></goexml>`, "duplicate option text"},
		{"id with space", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a b" text="x"/></window></goexml>`, "letters, digits or _"},
		{"id with slash", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="../x" text="x"/></window></goexml>`, "letters, digits or _"},
		{"declared namespace", `<x:goexml xmlns:x="urn:x" version="0.1.0"><x:window title="x" size="1x1"><x:label id="a" text="x"/></x:window></x:goexml>`, "namespaces are not supported"},
		{"undeclared namespace", `<x:goexml version="0.1.0"><x:window title="x" size="1x1"><x:label id="a" text="x"/></x:window></x:goexml>`, "namespaces are not supported"},
		{"dup attribute", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="one" text="two"/></window></goexml>`, "duplicate attribute"},
		{"table no rows", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"/></window></goexml>`, "at least one <row>"},
		{"table cell mismatch", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"><row><cell text="a"/><cell text="b"/></row><row><cell text="only"/></row></table></window></goexml>`, "has 1 cells, want 2"},
		{"table bad child", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"><label id="l" text="x"/></table></window></goexml>`, "not allowed inside <table>"},
		{"table bad cell child", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"><row><label id="l" text="x"/></row></table></window></goexml>`, "not allowed inside <row>"},
		{"table cell no text", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"><row><cell/></row></table></window></goexml>`, "needs text"},
		{"cell inner text both", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"><row><cell text="a">b</cell></row></table></window></goexml>`, "not both"},
		{"text both", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><text id="t" v="a">b</text></vbox></window></goexml>`, "not both"},
		{"text empty", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><text id="t"/></vbox></window></goexml>`, "needs text"},
		{"text bad wrap", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><text id="t" wrap="wide">x</text></vbox></window></goexml>`, "wrap must be"},
		{"label rejects wrap", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" wrap="word"/></window></goexml>`, "has no attribute"},
		{"expand removed", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" expand="true"/></window></goexml>`, "has no attribute"},
		{"style two widths", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" style="w-full w-full"/></window></goexml>`, "more than one w-*"},
		{"style two heights", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" style="h-2 h-full"/></window></goexml>`, "more than one h-*"},
		{"table empty row", `<goexml version="0.1.0"><window title="x" size="1x1"><table id="t"><row/></table></window></goexml>`, "at least one <cell>"},
		{"rect no color", `<goexml version="0.1.0"><window title="x" size="1x1"><rect id="r"/></window></goexml>`, "color must look like"},
		{"rect bad color", `<goexml version="0.1.0"><window title="x" size="1x1"><rect id="r" color="red"/></window></goexml>`, "color must look like"},
		{"rect bad hex", `<goexml version="0.1.0"><window title="x" size="1x1"><rect id="r" color="#zzzzzz"/></window></goexml>`, "color must look like"},
		{"rect children", `<goexml version="0.1.0"><window title="x" size="1x1"><rect id="r" color="#2e2e2e"><label id="l" text="x"/></rect></window></goexml>`, "cannot have children"},
		{"image bad src", `<goexml version="0.1.0"><window title="x" size="1x1"><image id="i" src="assets/logo final.png"/></window></goexml>`, "bad src"},
		{"image src url", `<goexml version="0.1.0"><window title="x" size="1x1"><image id="i" src="https://example.com/a.png"/></window></goexml>`, "bad src"},
		{"image src template", `<goexml version="0.1.0"><window title="x" size="1x1"><image id="i" src="{{name}}"/></window></goexml>`, "bad src"},
		{"image empty src", `<goexml version="0.1.0"><window title="x" size="1x1"><image id="i" src=""/></window></goexml>`, "bad src"},
		{"image slot without id", `<goexml version="0.1.0"><window title="x" size="1x1"><image fit="contain"/></window></goexml>`, "needs an id attribute"},
		{"vbox-expand without id", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand/></window></goexml>`, "needs an id"},
		{"text template without id", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><text>{{name}}</text></vbox></window></goexml>`, "needs an id attribute"},
		{"button template without id", `<goexml version="0.1.0"><window title="x" size="1x1"><button text="{{x}}" onClick="P"/></window></goexml>`, "needs an id attribute"},
		{"select template without id", `<goexml version="0.1.0"><window title="x" size="1x1"><select onChange="P"><option value="v" text="{{a}}"/></select></window></goexml>`, "needs an id attribute"},
		{"image bad fit", `<goexml version="0.1.0"><window title="x" size="1x1"><image id="i" src="a/b.png" fit="zoom"/></window></goexml>`, "fit must be"},
		{"vbox-expand children", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand id="f"><label id="l" text="x"/></vbox-expand></window></goexml>`, "takes no children"},
		{"vbox-expand scrollable value", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox-expand id="f" scrollable="true"/></window></goexml>`, "scrollable takes no value"},
		{"style unknown", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" style="wide"/></window></goexml>`, "unknown style"},
		{"style bad pad", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" style="p-9"/></window></goexml>`, "bad padding"},
		{"style bad margin", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x" style="m-9"/></window></goexml>`, "bad margin"},
		{"justify bad", `<goexml version="0.1.0"><window title="x" size="1x1"><hbox id="b" justify="middle"><label id="a" text="x"/></hbox></window></goexml>`, "justify must be"},
		{"justify on label", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x" justify="center"/></window></goexml>`, "has no attribute"},
		{"grid bad mode", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="g" column="masonry"><label id="a" text="x"/></vbox></window></goexml>`, "unknown column mode"},
		{"grid min without grid", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="g" min="2"><label id="a" text="x"/></vbox></window></goexml>`, "min needs column"},
		{"grid min exceeds max", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="g" column="grid" min="5" max="2"><label id="a" text="x"/></vbox></window></goexml>`, "min 5 exceeds max 2"},
		{"grid min zero", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="g" column="grid" min="0"><label id="a" text="x"/></vbox></window></goexml>`, "min must be an integer"},
		{"trailing text", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x"/></window></goexml>garbage`, "text outside root element"},
		{"second root", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x"/></window></goexml><goexml version="0.1.0"><window title="y" size="1x1"><label id="b" text="y"/></window></goexml>`, "after root element"},
		{"wrap names off", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><text id="t" wrap="wide">x</text></vbox></window></goexml>`, "word, break or off"},
		{"table fixed height", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><table id="t" style="h-2"><row><cell text="A"/></row></table></vbox></window></goexml>`, "not supported on a table"},
		{"table full height", `<goexml version="0.1.0"><window title="x" size="1x1"><vbox id="b"><table id="t" style="h-full"><row><cell text="A"/></row></table></vbox></window></goexml>`, "not supported on a table"},
		{"window two children", `<goexml version="0.1.0"><window title="x" size="1x1"><rect id="a" color="#111111"/><rect id="b" color="#222222"/></window></goexml>`, "exactly one child"},
		{"border region two children", `<goexml version="0.1.0"><window title="x" size="1x1"><border id="bd"><top><label id="a" text="a"/><label id="b" text="b"/></top><center><label id="c" text="c"/></center></border></window></goexml>`, "at most one child"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := Decode([]byte(c.doc))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err, c.want)
			}
		})
	}
}

// Errors name line, tag, and the element's id when it has one.
func TestDecodeErrorNamesID(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<label id="lblTitle" text="x" color="red"/></window></goexml>`
	_, err := Decode([]byte(doc))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, want := range []string{`line 1`, `<label`, `id="lblTitle"`, `has no attribute`} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}

	// A node without an id keeps the plain <tag> form.
	doc = `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="b"><option text="A"/></vbox></window></goexml>`
	_, err = Decode([]byte(doc))
	if err == nil || !strings.Contains(err.Error(), "<option> is not allowed inside <vbox>") {
		t.Fatalf("error %q does not use the plain <option> form", err)
	}
}

// Unknown attributes are reported deterministically: sorted and all of
// them, never one at random.
func TestDecodeUnknownAttrsDeterministic(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<label id="a" text="x" zeta="1" alpha="2" mu="3"/></window></goexml>`
	var first string
	for i := 0; i < 50; i++ {
		_, err := Decode([]byte(doc))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if first == "" {
			first = err.Error()
		} else if err.Error() != first {
			t.Fatalf("message changed across runs: %q vs %q", first, err)
		}
	}
	for _, want := range []string{`"alpha"`, `"mu"`, `"zeta"`} {
		if !strings.Contains(first, want) {
			t.Fatalf("message %q does not name %q", first, want)
		}
	}
	if strings.Index(first, `"alpha"`) > strings.Index(first, `"zeta"`) {
		t.Fatalf("message %q is not sorted", first)
	}
}

// color and size errors carry the tag and id like every other error.
func TestDecodeColorSizeNameNode(t *testing.T) {
	_, err := Decode([]byte(`<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<rect id="r1" color="red"/></window></goexml>`))
	if err == nil || !strings.Contains(err.Error(), `<rect id="r1"> color must look like`) {
		t.Fatalf("color error %q does not name the node", err)
	}
	_, err = Decode([]byte(`<goexml version="0.1.0"><window title="x" size="bad">` +
		`<rect id="r1" color="#2e2e2e"/></window></goexml>`))
	if err == nil || !strings.Contains(err.Error(), `<window> size must look like`) {
		t.Fatalf("size error %q does not name the node", err)
	}
}

// Directives and processing instructions are rejected inside an
// element instead of being silently dropped. The prolog keeps
// accepting them (a leading <?xml ...?> is fine).
func TestDecodeRejectsDirectiveAndPI(t *testing.T) {
	for _, doc := range []string{
		`<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<vbox id="b"><!DOCTYPE fool><label id="l" text="x"/></vbox></window></goexml>`,
		`<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<vbox id="b"><?php echo ?><label id="l" text="x"/></vbox></window></goexml>`,
	} {
		_, err := Decode([]byte(doc))
		if err == nil {
			t.Fatal("expected error for directive/PI inside an element, got nil")
		}
		if !strings.Contains(err.Error(), "not allowed inside an element") {
			t.Fatalf("error %q does not reject the directive/PI", err)
		}
	}
	root, err := Decode([]byte(`<?xml version="1.0"?>` +
		`<goexml version="0.1.0"><window title="x" size="1x1"><label id="l" text="x"/></window></goexml>`))
	if err != nil {
		t.Fatalf("prolog PI should be accepted: %v", err)
	}
	if root.Children[0].Attr("title") != "x" {
		t.Fatal("document with prolog PI did not decode")
	}
}

// Syntax errors report one line number — the syntax error's own line —
// and never the "inside:" pairing of two different lines.
func TestDecodeSyntaxErrorSingleLine(t *testing.T) {
	doc := "<goexml version=\"0.1.0\">\n  <window title=\"x\" size=\"1x1\">\n" +
		"    <label id=\"a\" text=\"x\">\n  </vbox>\n</window></goexml>"
	_, err := Decode([]byte(doc))
	if err == nil {
		t.Fatal("expected syntax error, got nil")
	}
	msg := err.Error()
	if strings.Contains(msg, "inside:") {
		t.Fatalf("syntax error %q still uses the inside: form", err)
	}
	if !strings.Contains(msg, "line 4: element <label> closed by </vbox>") {
		t.Fatalf("syntax error %q does not use the syntax error's own line", err)
	}
	if strings.Count(msg, "line") != 1 {
		t.Fatalf("syntax error %q carries more than one line number", err)
	}
}

// container + slot decode rules: exactly one slot, no nesting, slots
// only inside containers.
func TestDecodeContainerSlot(t *testing.T) {
	// Valid: container as window child, slot deep in the subtree.
	doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<container id="page"><vbox id="f"><label id="h" text="H"/><slot/><label id="ft" text="F"/></vbox></container>` +
		`</window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("valid container rejected: %v", err)
	}
	if got := len(root.Children[0].Children); got != 1 {
		t.Fatalf("container children = %d, want 1", got)
	}

	// Sibling containers are fine.
	doc = `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="b"><container id="a"><slot/></container><container id="c"><slot/></container></vbox>` +
		`</window></goexml>`
	if _, err := Decode([]byte(doc)); err != nil {
		t.Fatalf("sibling containers rejected: %v", err)
	}

	for _, tc := range []struct{ name, doc, want string }{
		{"nested container", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="outer"><container id="inner"><slot/></container></container></window></goexml>`,
			"cannot be nested inside <container>"},
		{"container deep in nested container", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="outer"><vbox id="f"><container id="inner"><slot/></container></vbox></container></window></goexml>`,
			"cannot be nested inside <container>"},
		{"zero slots", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="p"><vbox id="f"><label id="l" text="x"/></vbox></container></window></goexml>`,
			"needs exactly one <slot>"},
		{"two slots", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="p"><vbox id="f"><slot/><slot/></vbox></container></window></goexml>`,
			"needs exactly one <slot>"},
		{"orphan slot in window", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<slot/></window></goexml>`,
			"only allowed inside a <container>"},
		{"orphan slot in vbox", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<vbox id="b"><slot/></vbox></window></goexml>`,
			"only allowed inside a <container>"},
		{"container two children", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="p"><label id="a" text="a"/><label id="b" text="b"/></container></window></goexml>`,
			"needs exactly one child"},
		{"container without id", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container><slot/></container></window></goexml>`,
			"needs an id"},
		{"slot with background", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="p"><slot style="bg-[#112233]"/></container></window></goexml>`,
			"renders no visual layer"},
		{"slot with border and rounding", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<container id="p"><slot style="w-full h-2 b-2 b-[#445566] rounded-md"/></container></window></goexml>`,
			"renders no visual layer"},
	} {
		_, err := Decode([]byte(tc.doc))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}

	// Dimension-only styles on a slot stay valid.
	doc = `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<container id="p"><slot style="w-full h-2 m-1 p-2"/></container></window></goexml>`
	if _, err := Decode([]byte(doc)); err != nil {
		t.Fatalf("dimension-only slot style rejected: %v", err)
	}
}

// DecodeItem validates the fragment envelope with the same schema rules.
func TestDecodeItem(t *testing.T) {
	// Valid: item root with one container child.
	item := `<goexml-item version="0.1.0"><container id="body"><slot/></container></goexml-item>`
	root, err := DecodeItem([]byte(item))
	if err != nil {
		t.Fatalf("valid fragment rejected: %v", err)
	}
	if root.Children[0].Type != "container" {
		t.Fatalf("fragment child = %s, want container", root.Children[0].Type)
	}

	for _, tc := range []struct{ name, item, want string }{
		{"document envelope", `<goexml version="0.1.0"><window title="x" size="1x1"><label id="a" text="x"/></window></goexml>`,
			"fragment root must be <goexml-item>"},
		{"two children", `<goexml-item version="0.1.0"><label id="a" text="a"/><label id="b" text="b"/></goexml-item>`,
			"must hold exactly one child"},
		{"window inside", `<goexml-item version="0.1.0"><window title="x" size="1x1"><label id="a" text="x"/></window></goexml-item>`,
			"is not allowed inside <goexml-item>"},
		{"wrong version", `<goexml-item version="0.2.0"><label id="a" text="x"/></goexml-item>`,
			`unsupported version "0.2.0"`},
		{"nested nesting rule applies", `<goexml-item version="0.1.0"><container id="o"><container id="i"><slot/></container></container></goexml-item>`,
			"cannot be nested inside <container>"},
	} {
		_, err := DecodeItem([]byte(tc.item))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}
}

// Placeholder tokens in dynamic text attributes are validated at decode.
func TestDecodePlaceholders(t *testing.T) {
	// Valid tokens pass on label and text v.
	doc := `<goexml version="0.1.0"><window title="x" size="1x1">` +
		`<vbox id="b"><label id="g" text="Halo {{name}}, {{n}}"/><text id="t" v="{{body}}"/></vbox></window></goexml>`
	if _, err := Decode([]byte(doc)); err != nil {
		t.Fatalf("valid placeholders rejected: %v", err)
	}
	for _, tc := range []struct{ name, doc, want string }{
		{"unclosed", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<label id="g" text="Halo {{name"/></window></goexml>`,
			"unclosed placeholder"},
		{"bad token chars", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<label id="g" text="{{user name}}"/></window></goexml>`,
			`invalid placeholder {{user name}}`},
		{"empty token", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<label id="g" text="{{}}"/></window></goexml>`,
			`invalid placeholder {{}}`},
		{"entry text", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<entry id="e" text="{{x"/></window></goexml>`,
			"unclosed placeholder"},
		{"button text", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<button id="b" text="{{x" onClick="Go"/></window></goexml>`,
			"unclosed placeholder"},
		{"option text", `<goexml version="0.1.0"><window title="x" size="1x1">` +
			`<select id="s" onChange="P"><option value="1" text="{{x"/></select></window></goexml>`,
			"unclosed placeholder"},
	} {
		_, err := Decode([]byte(tc.doc))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: error %q does not contain %q", tc.name, err, tc.want)
		}
	}
}
