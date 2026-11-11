// Command demo renders an example goexml document with Fyne.
package main

import (
	"log"

	"fyne.io/fyne/v2/app"

	"github.com/five-zombie/goexmlfyne"
)

func main() {
	goexml.BasePath("examples")
	root, err := goexml.Read("form.xml")
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	act := goexml.BuildAct(map[string]func(){
		"OnCancel": func() { log.Println("demo: cancel") },
		"OnSave":   func() { log.Println("demo: save") },
	})
	win, err := root.ApplyAct(act)
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	a := app.New()
	w := a.NewWindow(win.Title)
	w.Resize(win.Size)
	w.SetContent(win.Content)
	w.ShowAndRun()
}
