# goexml-fyne

`goexml-fyne` renders declarative `goexml` XML documents into Fyne widgets.
Write the layout in XML, keep the logic in Go. Implements the `goexml 0.1.0` standard.

## Example

```xml
<goexml version="0.1.0">
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
</goexml>
```

## Elements

- `window`: one per file; `title` and `size` (e.g. `800x600`).
- `vbox`, `hbox`: vertical / horizontal containers with `padding`,
  `spacing`, `align`, and optional `justify`. A `vbox` with
  `column="grid"` becomes a responsive grid clamped by `min`/`max`.
- `border`: uses `top`, `bottom`, `left`, `right`, `center` children.
- `label text="..."`: one line, no wrapping, no children.
- `button text onClick`: `onClick` names a Go function, no inline code.
- `entry text placeholder`: single-line input, disableable.
- `text v="..."` or `<text>body</text>`: long-form block. The `v`
  attribute and inner text are exclusive; one of them is required.
  Wraps words by default; `wrap="break"` breaks inside words and
  `wrap="off"` keeps one line. Inside a vertical box a text block
  fills the container width and grows to its full wrapped height.
  Inside a horizontal box wrapping texts split the leftover width
  equally; `spacing` stays the gutter between the columns.
- `select onChange` with `<option value text selected?>` children:
  `selected=""` (no value) marks the chosen option; `SelectValue(id)`
  returns the `value` of the current choice.
- `table onSelect` with `<row><cell>` children: first row is the
  header. Wrapped in a sizing box so the header plus every data row
  stays visible; each column is sized to its widest cell. `Find(id)`
  returns the `*widget.Table`.
- `rect color="#rrggbb"`: a color box for lines and divider bands.
- `cell text="..."` or `<cell>body</cell>`: same exclusive rule as
  `<text>`.

Every widget carries `id` (unique, `[A-Za-z0-9_]+`), `visible`
(true/false), and — where Fyne supports it (`button`, `entry`,
`select`) — `disabled` (true/false).

## Style

The `style` attribute works on every widget, space separated, in the
spirit of Tailwind:

- `w-full` / `h-full`: fill the parent width / height. Both resolve
  against the parent's content area — the parent's padding is never
  overlapped. `style="w-full h-full"` fills both axes.
- `h-N`: an exact content height of N*4 px (`h-2` = 8 px) whatever the
  widget contains. `h-0` is a real zero height: an invisible element
  that still takes part in layout through its margin — a spacer.
- Padding: `p-N`, `px-N`, `py-N`, `pt-N`, `pr-N`, `pb-N`, `pl-N`.
- Margin: `m-N`, `mx-N`, `my-N`, `mt-N`, `mr-N`, `mb-N`, `ml-N`.
- N runs 0-4 (4 px per step); specific sides win, order does not matter
  (`p-2 pt-4` = 16 px top, 8 px elsewhere).

A tag carries at most one `w-*` and one `h-*` class. Unknown utilities
are rejected by the parser. In a grid, the cell height is the row
height: `h-N` only raises the row minimum.

## Use

```go
import (
    "log"

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

The full element, attribute, and style reference lives in
[docs/goexml-standard-0.1.0.md](docs/goexml-standard-0.1.0.md).

## License

Apache-2.0, see [LICENSE](LICENSE).
