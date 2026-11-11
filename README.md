# goexml-fyne

`goexml-fyne` renders declarative `goexml` XML documents into Fyne widgets.
Write the layout in XML, keep the logic in Go. Implements the `goexml 0.1.0` standard.

## Example

```xml
<goexml version="0.1.0">
  <window title="Form" size="800x600">
    <vbox id="main" padding="8" spacing="4" align="start">
      <label id="lblTitle" text="Enter name" />
      <entry id="inpName" text="" placeholder="Name" expand="true" />
      <hbox id="actions" spacing="4" align="end">
        <button id="btnCancel" text="Cancel" onClick="OnCancel" />
        <button id="btnSave" text="Save" onClick="OnSave" />
      </hbox>
    </vbox>
  </window>
</goexml>
```

## Elements

- `<label text="...">`: one line, no wrapping, no children.
- `<text v="...">` or `<text>body</text>`: long-form block. The `v`
  attribute and inner text are exclusive; one of them is required.
  Wraps words by default; `wrap="break"` breaks inside words and
  `wrap="off"` keeps one line. Inside a vertical box a text block
  fills the container width and grows to its full wrapped height.
  Inside a horizontal box wrapping texts split the leftover width
  equally; `spacing` stays the gutter between the columns.
- `w-full` on a container fills the parent width and grows to its
  content height, grown text rows included.
- `<cell text="...">` or `<cell>body</cell>`: same exclusive rule as
  `<text>`; each column is sized to its widest cell.
- `<table>`: wrapped in a sizing box so the header plus every data
  row stays visible. `Find(id)` still returns the `*widget.Table`.

## Use

```go
import (
    "fyne.io/fyne/v2/app"

    "github.com/five-zombie/goexmlfyne"
}

// Standard Fyne order: the app exists before any widget is built.
a := app.New()
goexml.BasePath("examples")

root, err := goexml.Read("form.xml")
if err != nil {
    log.Fatal(err)
}

act := goexml.BuildAct(map[string]func(){
    "OnCancel": func() { log.Println("cancel") },
    "OnSave":   func() { log.Println("save") },
})

win, err := root.ApplyAct(act)
if err != nil {
    log.Fatal(err)
}

w := a.NewWindow(win.Title)
w.Resize(win.Size)
w.SetContent(win.Content)
w.ShowAndRun()
```

## Commands

- `go build ./...` builds everything including the demo.
- `go test ./...` runs the test suite.
- `go run ./cmd/demo` runs the showcase with every widget (needs a display).
- `go run ./cmd/demo form.xml` runs another file from `examples/`.

Requires Go 1.27. On Linux, Fyne needs a compiler and graphics
headers, e.g. `gcc`, `libgl1-mesa-dev`, and `xorg-dev` on Debian/Ubuntu.

## License

Apache-2.0, see [LICENSE](LICENSE).
