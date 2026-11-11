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
		{"missing id", `<goexml version="0.1.0"><window title="x" size="1x1"><label text="x"/></window></goexml>`, "needs an id"},
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
