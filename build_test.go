package goexml

import (
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestMain(m *testing.M) {
	a := test.NewApp()
	code := m.Run()
	a.Quit()
	os.Exit(code)
}

func buildValid(t *testing.T, tap map[string]func()) *Window {
	t.Helper()
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, tap)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	return w
}

func TestBuildForm(t *testing.T) {
	saved := false
	w := buildValid(t, map[string]func(){
		"OnBatal":  func() {},
		"OnSimpan": func() { saved = true },
	})
	if w.Title != "Form" {
		t.Fatalf("title = %q, want Form", w.Title)
	}
	if w.Size != fyne.NewSize(800, 600) {
		t.Fatalf("size = %v, want 800x600", w.Size)
	}
	box, ok := w.Content.(*fyne.Container)
	if !ok {
		t.Fatalf("content is %T, want *fyne.Container", w.Content)
	}
	if len(box.Objects) != 3 {
		t.Fatalf("content children = %d, want 3", len(box.Objects))
	}
	actions, ok := box.Objects[2].(*fyne.Container)
	if !ok || len(actions.Objects) != 2 {
		t.Fatalf("actions row is %T with %d children, want container with 2",
			box.Objects[2], len(actions.Objects))
	}
	btn, ok := actions.Objects[1].(*widget.Button)
	if !ok {
		t.Fatalf("save control is %T, want *widget.Button", actions.Objects[1])
	}
	btn.OnTapped()
	if !saved {
		t.Fatal("tapping save did not call OnSimpan")
	}
}

func TestBuildUnknownHandler(t *testing.T) {
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	_, err = Build(root, map[string]func(){})
	if err == nil || !strings.Contains(err.Error(), "no handler") {
		t.Fatalf("expected no-handler error, got %v", err)
	}
}

func TestBuildState(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<vbox id="b"><label id="l" text="x" visible="false"/>` +
		`<entry id="e" disabled="true"/></vbox></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	box := w.Content.(*fyne.Container)
	if box.Objects[0].Visible() {
		t.Fatal("hidden label is visible")
	}
	entry, ok := box.Objects[1].(*widget.Entry)
	if !ok {
		t.Fatalf("second child is %T, want *widget.Entry", box.Objects[1])
	}
	if !entry.Disabled() {
		t.Fatal("disabled entry reports enabled")
	}
}

func TestBuildBorder(t *testing.T) {
	doc := `<goexml version="0.1.0"><window title="x" size="10x10">` +
		`<border id="b"><top><label id="t" text="top"/></top>` +
		`<center><label id="c" text="mid"/></center></border></window></goexml>`
	root, err := Decode([]byte(doc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	w, err := Build(root, map[string]func(){})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if _, ok := w.Content.(*fyne.Container); !ok {
		t.Fatalf("content is %T, want *fyne.Container", w.Content)
	}
}

func TestBuildExampleFile(t *testing.T) {
	data, err := os.ReadFile("examples/form.xml")
	if err != nil {
		t.Fatalf("reading example: %v", err)
	}
	root, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if _, err := Build(root, map[string]func(){"OnBatal": func() {}, "OnSimpan": func() {}}); err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
}
