# Elements

Responsibility: the XML vocabulary — every element, its attributes,
children rules and value formats. This is the reference for what
`Decode` accepts; the normative source is
[goexml-standard-0.1.0.md](goexml-standard-0.1.0.md). Styling is in
[style.md](style.md), layout behavior in [layout.md](layout.md).

## Envelopes

- `<goexml version="0.1.0">` — the document root. Holds exactly one
  `<window>`.
- `<goexml-item version="0.1.0">` — the fragment root for runtime
  injection. Holds exactly one child widget, no `<window>`. See
  [injection.md](injection.md) and [lists.md](lists.md).

## Window

`<window title>` — `title` is required. The window's size is set
from Go (`Resize` on the `fyne.Window` after `Build`); without it the
toolkit default applies. Holds exactly one child; wrap several
widgets in a `<vbox>`.

## Structure elements

| Element | Attributes | Children |
|---------|-----------|----------|
| `vbox` | `spacing`, `align`, `justify`, `column`, `min`, `max` | 1+ widgets; vertical stack or grid |
| `hbox` | `spacing`, `align`, `justify` | 1+ widgets; horizontal stack |
| `vbox-expand` | `spacing`, `align`, `justify`, `column`, `min`, `max`, `scrollable` | **none** — items come from Go |
| `container` | — (`id` required) | exactly one child whose subtree holds exactly one `<slot>` |
| `slot` | `style` (dimension utilities only) | none — a leaf hole filled by Go; exactly one per `container` |

Notes:

- `column="grid"` turns a `vbox` (and `vbox-expand`) into a responsive
  column grid clamped by `min`/`max` (both need `column="grid"`, integers
  ≥ 1, `min ≤ max`).
- `vbox-expand` rejects children at decode — a child would never reach
  the screen. See [lists.md](lists.md).
- Containers never nest inside one container within a file; sibling
  containers are fine.

## Leaf widgets

| Element | Attributes | Text rule |
|---------|-----------|-----------|
| `label` | `text` | `text` attribute required |
| `text` | `v`, `wrap` | `v` attribute or inner text, not both; `wrap` = `word` (default), `break`, `off` |
| `button` | `text`, `onClick`, `disabled` | `text` + handler name required |
| `entry` | `text`, `placeholder`, `disabled` | — |
| `rect` | `color` | `color` is `#rrggbb` |
| `image` | `src`, `fit` | `src` optional — see below |
| `select` | `onChange`, `disabled` | only `<option>` children |
| `table` | `onSelect` | only `<row>` children |

- `option` — `value` and `text` required, unique text per select. The
  chosen marker is the bare `selected` attribute (or `selected="true"`);
  any other value is rejected. At most one chosen option.
- `row` — 1+ `<cell>`; every row must hold the same cell count.
- `cell` — `text` attribute or inner text, not both. The first row is
  the header.
- `image` — with `src`, the key must exist in the build's resource set
  and its bytes must decode as an image. Without `src`, the image is a
  runtime-filled slot and requires an `id`. `fit` is `contain`
  (default), `cover` or `fill`. The visual layer's rules are in
  [style.md](style.md), the image lifecycle in [runtime.md](runtime.md).

## Common attributes

`id`, `visible`, `style` are allowed on every widget element.
`disabled` is per tag: `button`, `entry`, `select` only.

- `id` — optional. Unique per file when present, `[A-Za-z0-9_]+`.
  Required for `container` and `vbox-expand` (Go must reference them),
  for an `image` without `src` (a slot image must be addressable), and
  for any element carrying a `{{placeholder}}` template (templates
  render through the setters, which address by id).
- `visible` — `true` (default) or `false`. A hidden element collapses
  its layout slot.
- `disabled` — `true` or `false`, default `false`.
- `style` — the style sub-language, see [style.md](style.md).

Elements that are never widgets — `goexml`, `goexml-item`, `window`
— take no common attributes. `slot` accepts only
`style`, and only dimension utilities (see [style.md](style.md)).

## Flag attributes

`scrollable` (on `vbox-expand`) and `selected` (on `option`) are flags:
attributes whose presence alone means true. The decoder rewrites the
bare form to `="true"` before parsing, so `<vbox-expand id="feed"
scrollable />` and `scrollable="true"` spell the same thing. The older
empty form (`scrollable=""`) and any other value are rejected with a
migration hint naming the accepted spellings.

## Attribute values at a glance

| Attribute | Accepted values |
|-----------|----------------|
| `visible`, `disabled` | `true`, `false` |
| `spacing` | integer ≥ 0 (pixels) |
| `align` | `start`, `center`, `end` |
| `justify` | `start`, `center`, `end`, `space-between`, `space-around`, `space-evenly` |
| `wrap` | `word`, `break`, `off` |
| `column` | `grid` |
| `min`, `max` | integer ≥ 1, only with `column="grid"`, `min ≤ max` |
| `color` | `#rrggbb` |
| `src` | resource key: letters, digits, `.`, `-`, `/`, `_` |
| `fit` | `contain`, `cover`, `fill` |

## Error format

Decode and build errors name the line, tag, and id when present:

```text
goexml: line 4: <button id="btnSave"> no handler for onClick "OnSave"
goexml: line 7: <vbox id="main"> padding moved to style: use the p-N utilities, e.g. p-2 for 8px
```

Removed attributes point at their migration (`padding` on `vbox`/`hbox`
moved to the `p-N` style utilities); unknown attributes are listed with
quotes and sorted for determinism.
