# Getting started

Responsibility: get a first goexml window on screen — install, a minimal
XML document, handlers, and running the demo. Element and style details
live in [elements.md](elements.md) and [style.md](style.md); the Go API
is referenced in [runtime.md](runtime.md).

## Install

```sh
go get github.com/five-zombie/goexmlfyne
```

Requires Go 1.27+ and Fyne v2 (`fyne.io/fyne/v2`). Fyne is the only
**direct** dependency; it brings its own indirect set (glfw, X11/DBus
bindings, text shaping) as usual for a GUI toolkit.

## The smallest document

`app.xml` (a real, runnable version of this lives in
[examples/form.xml](../examples/form.xml), cataloged in
[examples.md](examples.md)):

```xml
<goexml version="0.1.0">
  <window title="Hello">
    <vbox id="root" spacing="4" align="start">
      <label id="lblTitle" text="Hello" />
      <button id="btnClick" text="Click" onClick="OnClick" />
    </vbox>
  </window>
</goexml>
```

## Render it

```go
package main

import (
	"log"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/five-zombie/goexmlfyne"
)

func main() {
	a := app.New() // a Fyne app must exist before any widget is built
	handlers := map[string]func(){
		"OnClick": func() { log.Println("clicked") },
	}
	root, err := goexml.Read("app.xml")
	if err != nil {
		log.Fatal(err)
	}
	win, err := root.ApplyAct(goexml.BuildAct(handlers))
	if err != nil {
		log.Fatal(err)
	}
	w := a.NewWindow(win.Title)
	w.SetContent(container.NewScroll(win.Content))
	w.ShowAndRun()
}
```

Order matters: `app.New()` runs **before** `Read`/`ApplyAct`. Widget
construction inside `Build` measures text, which needs a running app.

## What just happened

1. `goexml.Read` loads and decodes the document (errors carry the source
   line, tag and id: `goexml: line 4: <button id="btnClick"> ...`).
2. `ApplyAct` renders the decoded tree into a `*goexml.Window`. Every
   handler named by `onClick`/`onChange`/`onSelect` must be present in
   the map — a missing name fails the build.
3. `win.Content` is the root Fyne object; `win.Title` names the
   window. The window's size is set from Go — call `Resize` on the
   `fyne.Window` after `Build` if you want a fixed size; without it
   the toolkit default applies.

## Relative paths

`Read` and `ReadItem` resolve relative paths against the directory set
by `goexml.BasePath`, so an app can keep its XML in one place:

```go
goexml.BasePath("ui") // Read("form.xml") reads ui/form.xml
```

Absolute paths are never rewritten.

## Run the demo

```sh
go run ./cmd/demo            # examples/showcase.xml
go run ./cmd/demo form.xml   # examples/form.xml
go run ./cmd/demo page.xml   # containers, fragment injection, templates
```

The demo resolves `examples/` through `BasePath` and registers the
showcase image payload in code — see [examples.md](examples.md).

## Next

- Vocabulary: [elements.md](elements.md)
- Styling: [style.md](style.md)
- Layout semantics: [layout.md](layout.md)
- Placeholders filled from Go: [injection.md](injection.md)
- Growing stacks: [lists.md](lists.md)
