// Command demo renders an example goexml document with Fyne.
// It shows showcase.xml by default, or the file named as the first
// argument: go run ./cmd/demo form.xml
package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

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
	root, err := goexml.Read(file)
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	act := goexml.BuildAct(map[string]func(){
		"OnCancel": func() { log.Println("demo: cancel") },
		"OnSave":   func() { log.Println("demo: save") },
		"OnPick":   func() { log.Println("demo: pick") },
		"OnRow":    func() { log.Println("demo: row") },
		"OnDemo":   func() { log.Println("demo: click") },
	})
	win, err := root.ApplyAct(act)
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	w := a.NewWindow(win.Title)
	w.Resize(win.Size)
	// Tall content (for example the ten thousand word showcase) does
	// not fit a real window, so scroll instead of letting Fyne clamp it.
	w.SetContent(container.NewScroll(win.Content))
	w.ShowAndRun()
}
