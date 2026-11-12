// Command demo renders an example goexml document with Fyne.
// It shows showcase.xml by default, or the file named as the first
// argument: go run ./cmd/demo form.xml
// go run ./cmd/demo page.xml also demonstrates containers, fragment
// injection, and {{placeholder}} templates.
package main

import (
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/five-zombie/goexmlfyne"
)

func main() {
	file := "showcase.xml"
	if len(os.Args) > 1 {
		file = os.Args[1]
	}
	// Standard Fyne order: the app exists before any widget is built.
	a := app.New()
	goexml.BasePath("examples")
	handlers := map[string]func(){
		"OnCancel": func() { log.Println("demo: cancel") },
		"OnSave":   func() { log.Println("demo: save") },
		"OnPick":   func() { log.Println("demo: pick") },
		"OnRow":    func() { log.Println("demo: row") },
		"OnDemo":   func() { log.Println("demo: click") },
		"OnGreet":  func() { log.Println("demo: greet") },
	}
	var win *goexml.Window
	if file == "page.xml" {
		w, err := buildPage(handlers)
		if err != nil {
			log.Fatalf("demo: %v", err)
		}
		win = w
	} else {
		root, err := goexml.Read(file)
		if err != nil {
			log.Fatalf("demo: %v", err)
		}
		w, err := root.ApplyAct(goexml.BuildAct(handlers))
		if err != nil {
			log.Fatalf("demo: %v", err)
		}
		win = w
	}
	w := a.NewWindow(win.Title)
	w.Resize(win.Size)
	// Tall content (for example the ten thousand word showcase) does
	// not fit a real window, so scroll instead of letting Fyne clamp it.
	w.SetContent(container.NewScroll(win.Content))
	w.ShowAndRun()
}

// buildPage renders the container example: the XML shell is built with
// a placeholder body, the page-body fragment replaces it through
// SetContent, and the greeting template renders from Go.
func buildPage(handlers map[string]func()) (*goexml.Window, error) {
	// The Greet button reads the entry and re-renders the template;
	// win is nil until Build returns, but handlers only fire later.
	var win *goexml.Window
	handlers["OnGreet"] = func() {
		name, _ := win.Text("inpName")
		if name == "" {
			name = "world"
		}
		if err := win.SetTextFormat("greeting", map[string]string{
			"name": name, "count": fmt.Sprint(greetCount),
		}); err != nil {
			log.Println("demo:", err)
		}
		greetCount++
	}
	root, err := goexml.Read("page.xml")
	if err != nil {
		return nil, err
	}
	// Build requires content for every slot: the placeholder holds the
	// place until the fragment replaces it, before the window shows.
	win, err = goexml.Build(root, handlers, map[string]fyne.CanvasObject{
		"page": widget.NewLabel(""),
	})
	if err != nil {
		return nil, err
	}
	item, err := goexml.ReadItem("page-body.xml")
	if err != nil {
		return nil, err
	}
	body, err := win.BuildItem(item, nil)
	if err != nil {
		return nil, err
	}
	win.Find("page").(*goexml.Container).SetContent(body)
	if err := win.SetTextFormat("greeting", map[string]string{
		"name": "world", "count": "0",
	}); err != nil {
		return nil, err
	}
	return win, nil
}

// greetCount counts Greet presses for the template's {{count}}.
var greetCount int
