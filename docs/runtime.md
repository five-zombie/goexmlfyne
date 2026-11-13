# Runtime API

Responsibility: the Go surface — building a window, the convenience
loaders, and every setter a built window exposes. The XML side is in
[elements.md](elements.md) and [style.md](style.md); row stacks have
their own page in [lists.md](lists.md).

## Building

```go
func Build(root *Node, tap map[string]func(),
    contents map[string]fyne.CanvasObject,
    resources map[string]fyne.Resource) (*Window, error)

func (n *Node) ApplyAct(act Act) (*Window, error)
func (n *Node) ApplyActContents(act Act, contents map[string]fyne.CanvasObject) (*Window, error)
func (n *Node) ApplyActResources(act Act, contents map[string]fyne.CanvasObject,
    resources map[string]fyne.Resource) (*Window, error)
func BuildAct(handlers map[string]func()) Act
```

- `tap` holds every handler named by `onClick`/`onChange`/`onSelect`;
  a missing name fails the build.
- `contents` fills every `<container>` slot keyed by container id — an
  unfilled slot or an unknown key fails the build.
- `resources` holds the named image payloads `<image src>` resolves
  against; a missing key or undecodable bytes fail the build. `nil`
  normalizes to an empty live set: an app that fills slot images at
  runtime adds to it through `win.SetImageRes(id, res)` (or swaps
  named keys with `win.SetImage(id, key)`) — the set itself is
  internal to the library, so external code reaches it only through
  these setters.
- `ApplyAct*` are the same three steps chained on a decoded tree; a
  document with `<container>` needs `ApplyActContents` or
  `ApplyActResources`.

```go
win, err := root.ApplyActResources(goexml.BuildAct(handlers), nil, res)
```

## Window

```go
type Window struct {
    Title   string           // from the XML
    Size    fyne.Size        // from the WxH attribute
    Content fyne.CanvasObject // the root Fyne object to show
}
```

- `Find(id)` returns the built object for an id, or nil. The concrete
  type depends on the element: `*widget.Label` for label/text,
  `*widget.Button`, `*widget.Entry`, `*widget.Select`, `*widget.Table`
  (the table itself, not its sizing wrapper), `*canvas.Image`,
  `*goexml.Container`, `*goexml.List`. Id-less elements are not
  registered and stay invisible to `Find` and the setters.

## Text and templates

- `SetText(id, v)` sets the text of a label, text, button or entry.
  Setting a plain text removes the `{{placeholder}}` template
  registered for the id, so later `SetTextFormat` calls are rejected —
  the last call wins.
- `Text(id)` returns the current text; an entry's value reflects what
  the user typed.
- `SetTextFormat(id, vals)` renders the id's `{{placeholder}}`
  template with `vals` and sets it. Repeatable; each call re-renders
  from the raw template.

## Selects

- `SetValue(id, v)` selects the option carrying value `v` (by value,
  not label).
- `SelectValue(id)` returns the current choice's `value` and `true`
  when the id is a select; an unknown id or a non-select returns
  `"", false`.
- `SetValueFormat(id, vals)` renders the `{{placeholder}}` templates
  in the option labels and updates the option list, keeping the
  current selection by value.

## Images

- `SetImage(id, key)` swaps the image's resource, resolving `key`
  against the resource set the build used (now including any merged
  fragment sets). The min size follows the new resource's intrinsic
  pixels.
- `SetImageRes(id, res)` attaches fetched bytes directly — no resource
  set lookup — for payloads that arrive late (server, database,
  object store).
- `UnfilledImages()` returns the sorted ids of slot images (no `src`)
  not yet filled, so an app can audit leftovers before showing.
- Both setters reject unknown ids, non-image nodes, unknown keys,
  nil resources and undecodable bytes — a filled image never renders
  as a silent empty box.

## Loaders

```go
func BasePath(dir string)             // prefix for relative Read paths
func Read(path string) (*Node, error)      // document
func ReadItem(path string) (*Node, error)  // fragment
```

`Read` decodes a document, `ReadItem` a fragment, in one call.
Relative paths resolve against `BasePath` when set; absolute paths are
never rewritten. Decoded trees can also be embedded at compile time —
see [codegen.md](codegen.md).

## Fragments and stacks

- `(*Window).BuildItem(node, contents, resources)` builds a
  `goexml-item` fragment into the window's id space — lifecycle and
  rebuild rules in [injection.md](injection.md).
- `Find("feed").(*goexml.List)` handles a `<vbox-expand>`: Draft,
  Commit, Clear — see [lists.md](lists.md).

## Error conventions

Every error names the source line, the tag, and the id when present;
handlers and setters use the same shape for runtime mistakes:

```text
goexml: line 4: <button id="btnSave"> no handler for onClick "OnSave"
goexml: no widget with id "lblGone"
goexml: unknown image resource "assets/logo.png"
```
