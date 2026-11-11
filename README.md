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

## Use

```go
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
```

## Commands

- `go build ./...` builds everything including the demo.
- `go test ./...` runs the test suite.
- `go run ./cmd/demo` runs the demo (needs a display).

Requires Go 1.27. On Linux, Fyne needs a compiler and graphics
headers, e.g. `gcc`, `libgl1-mesa-dev`, and `xorg-dev` on Debian/Ubuntu.

## License

Apache-2.0, see [LICENSE](LICENSE).
