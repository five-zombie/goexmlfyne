package gen

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/five-zombie/goexmlfyne"
)

func TestMain(m *testing.M) {
	a := test.NewApp()
	code := m.Run()
	a.Quit()
	os.Exit(code)
}

// testImage encodes a tiny valid PNG, so the showcase's <image> has
// decodable bytes to resolve in tests.
func testImage() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// sources maps each generated function to the XML file it came from.
// The list must stay in sync with the go:generate directive.
var sources = []struct {
	fn   func() *goexml.Node
	file string
}{
	{Form, "../examples/form.xml"},
	{Page, "../examples/page.xml"},
	{PageBody, "../examples/page-body.xml"},
	{FeedItem, "../examples/feed-item.xml"},
	{Showcase, "../examples/showcase.xml"},
	{Widgets, "../examples/widgets.xml"},
}

// Generated trees are byte-identical to what the runtime parser
// produces from the same file: codegen is only an alternative builder
// for the same validated trees, never a second schema.
func TestGeneratedEqualsDecoded(t *testing.T) {
	for _, s := range sources {
		data, err := os.ReadFile(s.file)
		if err != nil {
			t.Fatalf("reading %s: %v", s.file, err)
		}
		// The same envelope routing the generator used: documents
		// decode with Decode, fragments with DecodeItem.
		want, err := goexml.Decode(data)
		if err != nil {
			want, err = goexml.DecodeItem(data)
			if err != nil {
				t.Fatalf("decoding %s: %v", s.file, err)
			}
		}
		got := s.fn()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("generated %s differs from decoded tree", s.file)
		}
	}
}

// Generated pages build the same way decoded ones do: no XML at
// runtime, the usual Build contract applies unchanged.
func TestGeneratedBuilds(t *testing.T) {
	handlers := map[string]func(){
		"OnCancel": func() {}, "OnSave": func() {},
		"OnPick": func() {}, "OnRow": func() {},
		"OnDemo": func() {}, "OnGreet": func() {},
	}
	for _, s := range sources {
		root := s.fn()
		if s.file == "../examples/page-body.xml" {
			// A fragment builds into a host window, like any item.
			hostRoot, err := goexml.Decode([]byte(
				`<goexml version="0.1.0"><window title="host" size="1x1">` +
					`<label id="hostlbl" text="x"/></window></goexml>`))
			if err != nil {
				t.Fatal(err)
			}
			host, err := goexml.Build(hostRoot, handlers, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := host.BuildItem(root, nil, nil); err != nil {
				t.Fatalf("BuildItem %s: %v", s.file, err)
			}
			continue
		}
		if s.file == "../examples/feed-item.xml" {
			// A fragment appends into a <vbox-expand>, its ids
			// private to each item.
			hostRoot, err := goexml.Decode([]byte(
				`<goexml version="0.1.0"><window title="host" size="1x1">` +
					`<vbox-expand id="feed"/></window></goexml>`))
			if err != nil {
				t.Fatal(err)
			}
			host, err := goexml.Build(hostRoot, handlers, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			feed, ok := host.Find("feed").(*goexml.List)
			if !ok {
				t.Fatalf("Find(feed) = %T, want *goexml.List", host.Find("feed"))
			}
			row, err := feed.Append(root, nil, nil)
			if err != nil {
				t.Fatalf("Append %s: %v", s.file, err)
			}
			if err := row.SetTextFormat("rowTitle", map[string]string{"title": "gen"}); err != nil {
				t.Fatalf("SetTextFormat %s: %v", s.file, err)
			}
			if feed.Count() != 1 {
				t.Fatalf("Count = %d, want 1", feed.Count())
			}
			continue
		}
		var contents map[string]fyne.CanvasObject
		if s.file == "../examples/page.xml" {
			contents = map[string]fyne.CanvasObject{
				"page": widget.NewLabel(""),
			}
		}
		// The showcase carries an <image>: feed it the resource set
		// the demo would, so the generated tree builds the same way.
		resources := map[string]fyne.Resource(nil)
		if s.file == "../examples/showcase.xml" {
			resources = map[string]fyne.Resource{
				"assets/demo.png": fyne.NewStaticResource("demo.png", testImage()),
			}
		}
		w, err := goexml.Build(root, handlers, contents, resources)
		if err != nil {
			t.Fatalf("Build %s: %v", s.file, err)
		}
		if w.Content == nil {
			t.Fatalf("Build %s returned no content", s.file)
		}
	}
}
