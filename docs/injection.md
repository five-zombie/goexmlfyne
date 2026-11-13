# Injection

Responsibility: containers, slots and fragments — how an XML design
declares a hole that Go fills, and how a fragment joins (and re-joins)
a document's id space. The List alternative for repeated rows is in
[lists.md](lists.md); the API reference is [runtime.md](runtime.md).

## Containers and slots

```xml
<container id="page">
  <vbox id="shell" spacing="8">
    <label id="head" text="goexml page" />
    <slot style="h-full" />
    <label id="foot" text="footer stays in XML" />
  </vbox>
</container>
```

- A `container` holds free design (any widget) and **exactly one**
  `<slot>` somewhere in its subtree. Containers never nest inside one
  container within a file; sibling containers are fine.
- A `slot` is a leaf. Its `style` decides the area it claims
  (`w-full`, `h-N`, padding, margin); visual tokens are rejected — a
  slot is a hole, only its filled content shows.
- `Build` requires content for **every** container id, keyed by id.
  An unfilled slot fails the build (`slot must be filled`), and a
  contents key that matches no container also fails
  (`no container with id "x"`) — a typo on the Go side is an error,
  not a silent nothing.
- `Find(id)` on a built window returns the `*goexml.Container`;
  `SetContent(obj)` replaces the slot's content at any time, and
  passing `nil` empties the slot (so a build-time-required slot can
  never hold a nil object — an empty slot stays a refused hole, not a
  panic). The holder measures to the content's own minimum size and
  the content fills the area the parent's layout assigned.
- A container with `visible="false"` hides its whole wrapper, like
  any widget.

## Fragments (`goexml-item`)

```xml
<goexml-item version="0.1.0">
  <vbox id="body" style="p-1" spacing="4">
    <label id="greeting" text="Hello {{name}}, {{count}} messages" />
    <entry id="inpGuest" text="" placeholder="Who greets?" style="w-full" />
    <button id="btnGreet" text="Greet" onClick="OnGreet" />
  </vbox>
</goexml-item>
```

- The fragment envelope holds **exactly one** child widget (a window
  is not allowed inside). The same schema rules apply, and its ids
  join the document's global id space.
- Load with `goexml.ReadItem(path)` (resolves against `BasePath`) or
  `goexml.DecodeItem(data)`.

## Building a fragment into a window

```go
item, err := goexml.ReadItem("page-body.xml")
body, err := win.BuildItem(item, nil, nil) // contents, then resources
win.Find("page").(*goexml.Container).SetContent(body)
```

`BuildItem(node, contents, resources)`:

- `contents` fills every container slot **inside the fragment**, keyed
  by the container id — the same contract `Build` takes.
- `resources` is **merged into** the window's resource set: a fragment
  may name its own payloads and reuse the document's keys, and a nil
  argument contributes nothing (the document's set survives).
- The fragment must not collide with the document's own ids
  (`id "b" is already used by the document`); a fragment container
  must be filled (`slot must be filled`); a contents key the fragment
  lacks is rejected.

## Rebuild semantics (last built page wins)

Rebuilding a fragment over the same ids is the page-switch lifecycle:

- Ids the new fragment no longer declares become unreachable — their
  widgets, select value tables and templates are retired, so a widget
  from the previous page never outlives it in `Find`/`Text`/`SetValue`.
- Ids both fragments share are overwritten (last one wins).
- A widget that rebuilt **plain** (no `{{placeholder}}` anymore) must
  not inherit the previous page's template or option table: the stale
  per-id state is dropped.
- A failed `BuildItem` is atomic: the fragment builds into temporary
  maps and is merged only after every check passes — ids **and**
  resource payloads alike — so an error leaves nothing behind and a
  corrected retry just works.

## When to prefer a List

A container plus `BuildItem` replaces one hole with one page. For a
stack of many rows built from the same fragment — a feed, a list of
entries — use `<vbox-expand>` and the Draft/Commit flow instead:
[lists.md](lists.md).
