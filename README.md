# goexml-fyne

`goexml-fyne` renders declarative `goexml` XML documents into Fyne widgets.
Write the layout in XML, keep the logic in Go. Implements the `goexml 0.1.0` standard.

## Example

```xml
<goexml version="0.1.0">
  <window title="Form" size="800x600">
    <vbox id="main" style="p-2" spacing="4" align="start">
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

- `window`: one per file; `title`, `size` (e.g. `800x600`), and exactly
  one child — wrap multiple children in a `vbox` yourself.
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
  returns the `*widget.Table`. Height style utilities (`h-full`,
  `h-N`) are not supported: a table always sizes to its content.
- `rect color="#rrggbb"`: a color box for lines and divider bands.
- `vbox-expand scrollable?`: an append-only stack. The XML reserves
  the place; items are built at runtime from `goexml-item` fragments
  via `Find(id)` + `List.Append` — per-item slot contents and image
  resources, the fragment's ids private to each item (the same
  fragment stacks many times). `Clear` empties the stack and retires
  the handles. `scrollable` wraps the stack so `h-N`/`h-full` bound
  its height; without size style the stack sizes to its items. The
  visual layer applies like on every widget.
- `image src? fit?`: a picture. With `src`, it names a registered
  resource from the build's resource set (never a file path or URL;
  an unknown key or undecodable bytes fail the build). Without `src`
  it is a runtime-filled slot: the build leaves it empty, and the app
  must fill it before showing — `SetImage(id, key)` resolves a key
  from the resource set, `SetImageRes(id, res)` attaches fetched
  bytes directly, and `UnfilledImages()` reports slot images still
  unfilled. `fit` is `contain` (default), `cover` or `fill`. Without
  size style a filled image is as large as its pixels; `h-*` and
  `w-*` size it like any leaf. The visual layer draws behind the
  pixels (rounded does not clip them).
- `cell text="..."` or `<cell>body</cell>`: same exclusive rule as
  `<text>`.
- `container id`: an injection frame. It holds free design (any
  widget) and **exactly one** `<slot>` somewhere in its subtree.
  Containers never nest within one file; siblings are fine, and
  runtime composition through injection is unaffected.
- `slot`: the hole filled from Go. A leaf that accepts only `style`
  (`w-*`, `h-N`, padding, margin); its style decides the area it
  gets, and the injected object stretches to fill that area. `Build`
  requires content for every container id, and `SetContent` on the
  built `*Container` replaces the slot's content.
- `goexml-item`: the root of an injectable fragment, requires
  `version="0.1.0"`. It holds **exactly one** child widget (no
  window), the same schema rules apply, and its ids join the
  document's global id space. Build fragments with
  `(*Window).BuildItem`; a fragment id must not collide with the
  document's ids, while rebuilding a fragment replaces its ids and
  slot content (the last built page wins): ids the new fragment no
  longer declares become unreachable, and a widget that rebuilt plain
  loses the previous page's templates.

`id` (unique, `[A-Za-z0-9_]+`) is **optional**: give it to elements
Go addresses at runtime, and skip it on possibly-static ones (a plain
label, a `rect` divider, a layout box). It stays required on the
elements that cannot be static — `container` (slots are filled by id),
`vbox-expand` (the handle comes from `Find`), an `image` without `src`
(a slot image must be addressable), and any element carrying a
`{{placeholder}}` template (templates render through the setters,
which address by id). `visible` (true/false) is common, and so is
`disabled` (true/false) where Fyne supports it (`button`, `entry`,
`select`).

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

## Visual style

Boxes and widgets can carry a visual layer behind their content —
background, border and rounded corners. The layer is drawn inside the
element's bounds: it never moves or grows the content, and padding
stays the inner spacing on top of it.

- `bg-[#rrggbb]`: a solid background behind the whole element,
  including its padding area.
- `b-[#rrggbb]` sets the border color on all four sides;
  `bt-`/`br-`/`bl-`/`bb-[#rrggbb]` per side. `b-N` sets the border
  width on all four sides (0-16 px, direct pixels);
  `bt-`/`br-`/`bl-`/`bb-N` per side. A side draws only when it has
  both a width and a color — either alone is a decode error.
- `rounded` is a 4 px radius on all corners; `rounded-md` 8 px,
  `rounded-xl` 16 px, `rounded-N` N px (1-999). Per side:
  `rounded-t-*`/`b`/`l`/`r`; per corner: `rounded-tl-*`/`tr`/`bl`/`br`.
  A radius larger than half the element is clamped at render.
- Rounded corners with a border require a uniform border (`b-` only):
  a rounded stroke is one rectangle and cannot vary per side.

Duplicate `bg-` tokens are an error. All combinations that would
render nothing (a colorless width, a widthless color, `rounded`
without a background or border) are decode errors.

## Dynamic values

Static text stays in XML attributes. Values that change at runtime are
set from Go by id — there is no logic in XML:

- `SetText(id, v)` sets the text of a `label`, `text`, `button` or
  `entry`; `Text(id)` reads it back (an entry reflects what the user
  typed).
- `SetValue(id, v)` selects the `select` option carrying value `v`.
- `{{token}}` placeholders (letters, digits, underscore) may appear in
  the text attributes of `label`, `text`, `button`, `entry` and in
  `option` labels. Malformed tokens are rejected at decode. Before the
  first render the raw template shows.
- `SetTextFormat(id, vals)` re-renders the template from its raw XML
  text on every call. A plain `SetText` removes the template
  registration — the last call wins.
- `SetValueFormat(id, vals)` renders the option labels of a `select`
  and updates its option list; the current selection is kept by value.

## Use

```go
package main

import (
    "log"

    "fyne.io/fyne/v2/app"

    "github.com/five-zombie/goexmlfyne"
)

func main() {
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
}
```

For a document that uses `<container>`, build the body object first
and use the contents-aware variant instead of `ApplyAct`:

```go
package main

import (
    "log"

    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"

    "github.com/five-zombie/goexmlfyne"
)

func main() {
    a := app.New()
    goexml.BasePath("examples")

    root, err := goexml.Read("page.xml")
    if err != nil {
        log.Fatal(err)
    }

    act := goexml.BuildAct(nil)

    // Build requires content for every <container> slot; unknown keys
    // are errors.
    win, err := root.ApplyActContents(act, map[string]fyne.CanvasObject{
        "page": widget.NewLabel("injected body"),
    })
    if err != nil {
        log.Fatal(err)
    }

    w := a.NewWindow(win.Title)
    w.Resize(win.Size)
    w.SetContent(win.Content)
    w.ShowAndRun()
}
```

### Containers, fragments, and dynamic values

```go
// Build requires content for every <container> slot; unknown keys are
// errors. The placeholder below is replaced before the window shows.
win, err := goexml.Build(root, handlers, map[string]fyne.CanvasObject{
    "page": widget.NewLabel(""),
}, nil) // resources for <image src>; nil when the doc has no images

// A goexml-item fragment joins the window's id space.
item, err := goexml.ReadItem("page-body.xml")
body, err := win.BuildItem(item, nil, nil) // contents, then resources

// Replace the slot content without touching the surrounding design.
win.Find("page").(*goexml.Container).SetContent(body)

// Dynamic values by id.
win.SetText("head", "changed")                       // plain text
win.SetTextFormat("greeting", map[string]string{     // {{name}} template
    "name": "world", "count": "3",
})
name, _ := win.Text("inpName")                       // two-way entry
win.SetValue("choice", "opt2")                       // select by value
```

The full runnable version is `cmd/demo` with `examples/page.xml` and
`examples/page-body.xml`.

## Code generation

`goexmlgen` turns goexml files into Go source that constructs
`goexml.Node` trees, so the XML is validated at build time and ships
inside the binary — no parsing at runtime:

```sh
go run ./cmd/goexmlgen -out gen -pkg gen examples
```

- Every `.xml` file (documents and `goexml-item` fragments) is decoded
  with the same parser the runtime uses; any XML error fails the
  generation with the original file name and line, so broken XML is a
  compile error.
- The output is one bundle, `gen/goexmlgen.go`, with one deterministic
  function per input file named after it (`page-body.xml` →
  `PageBody()`). Ids must be unique across all input files of a run.
- Generated trees are backend-neutral and interchangeable with decoded
  ones: `goexml.Build(gen.Page(), handlers, contents)` builds exactly
  like a runtime-decoded page. Keep XML for pages that change at
  runtime, generated nodes for the ones that don't — both in the same
  window.
- CI regenerates and rejects any diff, so generated code is always up
  to date (see `gen/`, wired through `go:generate`).

## Commands

- `go build ./...` builds everything including the demo.
- `go test ./...` runs the test suite.
- `go run ./cmd/demo` runs the showcase with every widget (needs a display).
- `go run ./cmd/demo form.xml` runs another file from `examples/`.
- `go run ./cmd/demo page.xml` runs the container example: XML shell,
  fragment injection, and `{{placeholder}}` templates.
- `go run ./cmd/goexmlgen -out gen -pkg gen examples` regenerates the
  checked-in node trees in `gen/`.

Requires Go 1.27. On Linux, Fyne needs a compiler and graphics
headers, e.g. `gcc`, `libgl1-mesa-dev`, and `xorg-dev` on Debian/Ubuntu.

The full element, attribute, and style reference lives in
[docs/goexml-standard-0.1.0.md](docs/goexml-standard-0.1.0.md).

## License

Apache-2.0, see [LICENSE](LICENSE).
