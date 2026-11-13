# Lists (draft / commit)

Responsibility: the `<vbox-expand>` stack — building rows at runtime
from fragments through `List.Draft` + `List.Commit`, and the item
lifecycle. The XML side of the element is in
[elements.md](elements.md); the Go surface shared with windows is in
[runtime.md](runtime.md).

## The element

```xml
<vbox-expand id="feed" style="w-full h-24" spacing="2" scrollable />
```

The XML reserves the place only: the element takes **no children**
(decode rejects them — a child would never reach the screen). Items
are built in Go from `goexml-item` fragments. Layout attributes mirror
a static `vbox` — `spacing`, `align`, `justify`, and `column="grid"`
with `min`/`max` — and the bare `scrollable` wraps the stack in a
scroll so `h-N`/`h-full` bound its visible height; without size style
the stack sizes to its items. The visual layer applies like on every
widget.

## Draft → fill → Commit

```go
item, err := goexml.ReadItem("feed-item.xml")
for _, title := range titles {
    row, err := feed.Draft(item) // built, NOT yet on screen
    if err != nil {
        return err
    }
    if err := row.SetTextFormat("rowTitle", map[string]string{"title": title}); err != nil {
        return err
    }
    if err := feed.Commit(row); err != nil {
        return err
    }
}
```

- `Draft(item)` builds the fragment into an `*Item` that is not yet on
  the stack. Fill it at your own pace: `SetText`/`SetTextFormat`,
  `SetValue`, `Find(id)` + `Container.SetContent` for slots,
  `SetImage`/`SetImageRes` for images.
- `Commit(item)` stacks it at the end: it becomes visible and
  `Count()` grows by one.
- The item's ids are **private to the item** — the same fragment can
  be drafted many times without collision, and the window's `Find`
  never sees a row's ids.
- Handlers resolve against the window's handler map at draft time;
  image `src` keys resolve against the build's resource set, shared
  with the window.

## Commit refuses incomplete rows

`Commit` is the completeness gate:

- Every container slot in the fragment must be filled first — a
  committed row is complete on screen. An unfilled slot is an error:
  `slot "rowSlot" is not filled: set it before Commit`.
- A slot **image** without `src` may stay pending: it keeps its size
  and is audited by `UnfilledImages()` instead.
- Committing twice (`this item is already committed`), committing a
  cleared item (`this item was cleared by Clear`), or committing an
  item drafted by another list is an error.

A failed commit appends nothing; the item stays a draft and can be
completed and committed after the fix.

## Clear and retirement

`Clear()` drops every committed item and retires every draft. Handles
of cleared items are retired: a setter on one is an explicit error
(`goexml: this item was cleared by Clear`), `Find` returns nil and
`UnfilledImages` returns nothing — never a silent write into a
detached widget. The list itself keeps working after a clear.

## Item API

An `*Item` mirrors every window setter, scoped to itself:

`SetText`, `Text`, `SetValue`, `SelectValue`, `SetTextFormat`,
`SetValueFormat`, `SetImage`, `SetImageRes`, `UnfilledImages`,
`Find`. The window's setters never reach into an item, and an item's
setters never reach the window.

## Choosing between container and list

- One hole, one page → `<container>` + `BuildItem`
  ([injection.md](injection.md)).
- Many rows from one fragment → `<vbox-expand>` + Draft/Commit.
