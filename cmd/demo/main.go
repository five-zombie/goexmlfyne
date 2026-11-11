// Command demo renders an example goexml document with Fyne.
package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"

	"github.com/five-zombie/goexmlfyne"
)

func main() {
	data, err := os.ReadFile("examples/form.xml")
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	root, err := goexml.Decode(data)
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	win, err := goexml.Build(root, map[string]func(){
		"OnBatal":  func() { log.Println("demo: batal") },
		"OnSimpan": func() { log.Println("demo: simpan") },
	})
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	a := app.New()
	w := a.NewWindow(win.Title)
	w.Resize(win.Size)
	w.SetContent(win.Content)
	w.ShowAndRun()
}
