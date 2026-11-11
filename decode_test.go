package goexml

import (
	"strings"
	"testing"
)

const validDoc = `<goexml version="0.1.0">
  <window title="Form" size="800x600">
    <vbox id="main" padding="8" spacing="4" align="start">
      <label id="lblJudul" text="Isi nama" />
      <entry id="inpNama" text="" placeholder="Nama" expand="true" />
      <hbox id="actions" spacing="4" align="end">
        <button id="btnBatal" text="Batal" onClick="OnBatal" />
        <button id="btnSimpan" text="Simpan" onClick="OnSimpan" />
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
