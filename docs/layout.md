# Layout

Responsibility: how the layout engine sizes and positions elements —
box stacks, grids, border regions, scroll and the fill/exact-height
hints. The attribute lists per element are in
[elements.md](elements.md); the utility grammar is in
[style.md](style.md).

## Box stacks (`vbox`, `hbox`)

`vbox` stacks children vertically, `hbox` horizontally. In XML:

```xml
<vbox id="panel" spacing="4" align="center" justify="space-between">
  <label id="head" text="Title" />
  <button id="btnOk" text="OK" />
</vbox>
```

The engine ignores theme padding, so a document renders the same on
every platform; inner spacing is a style concern (the `p-N` utilities
wrap the box).

- `spacing` is the **minimum** gap between children in pixels.
- `align` is a **box-level** attribute: it positions the whole group of
  children on the cross axis (`start`, `center`, `end`). On a `vbox`
  the cross axis is horizontal, so `align="end"` makes the children hug
  the parent's right edge; on an `hbox` the cross axis is vertical, so
  `align="end"` hugs the bottom edge. It is not a per-child attribute.
- `justify` distributes **leftover** main-axis space on top of the
  minimum gaps:
  - `start` (default), `center`, `end` — a leading offset.
  - `space-between` — leftover shared evenly between children.
  - `space-around` — half-space around each child.
  - `space-evenly` — one even slot before, after and between.

Hidden children are skipped by the layout and their slot collapses;
the per-child hints stay aligned with the visible children.

## Per-child sizing hints

| Style | Box behavior |
|-------|--------------|
| `w-full` | a `vbox` child takes the parent's full width; an `hbox` child shares the leftover width equally with other filling siblings |
| `h-full` | a `vbox` child takes the leftover height after the fixed-size siblings; an `hbox` child fills the box's height (the cross axis), not the leftover |
| `h-N` | the child's content is exactly `N*4` px tall; the wrappers around it (padding, margin) are added on top so the content itself stays at `N*4` |
| wrapping `text` | re-measured after being wrapped at the assigned width, so the row grows to the true wrapped height |

Inside a vertical box a wrapping `text` fills the container width and
grows to its full wrapped height. Inside a horizontal box, wrapping
texts split the leftover width equally; `spacing` stays the gutter
between the columns.

## Grid columns

`column="grid"` on a `vbox` (or `vbox-expand`) flows children into
columns instead of one stack:

```xml
<vbox id="cards" column="grid" min="1" max="3" spacing="4">
  <vbox id="cardA" style="bg-[#1e293b] rounded-md p-2">…</vbox>
  <vbox id="cardB" style="bg-[#1e293b] rounded-md p-2">…</vbox>
  <vbox id="cardC" style="bg-[#1e293b] rounded-md p-2">…</vbox>
</vbox>
```

- `min` and `max` clamp the column count (integers ≥ 1, `min ≤ max`,
  both only legal with `column="grid"`).
- Column count adapts to the available width: the count comes from the
  widest child's minimum width clamped into `[min, max]` and the child
  count, so more room means more columns (up to `max`) and narrow
  space collapses toward `min`.
- The layout does **not** consult per-cell fill hints: every cell is
  resized to the shared column width, so all columns are equal.
- Row heights: each row is as tall as its tallest child (an `h-N` hint
  sets that height exactly), wrapped `text` grows the row to its true
  wrapped height, and leftover window height is shared equally by all
  rows.

## Border regions

`<border>` uses `top`, `bottom`, `left`, `right`, `center` children;
each region holds at most one widget. Multiple widgets in one region
must be wrapped in a `<vbox>` (decode rejects >1 child per region).
The center region fills the leftover space.

## Scroll

Scrolling is explicit, in one of two places:

- `<vbox-expand … scrollable>` wraps the stack in a scroll so `h-N` /
  `h-full` on the element bound the visible height; without size style
  the stack simply sizes to its items.
- In Go, a window whose content is taller than its window is shown
  through `container.NewScroll(win.Content)` — the demo does this, so
  tall content scrolls instead of Fyne clamping the window.

## Injection frames

`<container>` measures to its content's own minimum size (zero while
the slot is empty) and lays the injected object out to fill the area
the parent's layout assigned — the slot's own `style` decides that
area. Details in [injection.md](injection.md).

## Tables

A `table` always sizes to its content: the wrapper measures the header
plus every data row and sizes each column to its widest cell (plus
gutter). `h-full` and `h-N` are rejected on a table, because an exact
or filling height would overflow the slot and overlap siblings.
