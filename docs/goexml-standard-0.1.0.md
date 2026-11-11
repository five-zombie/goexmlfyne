# The goexml 0.1.0 standard

The `goexml` standard is the XML format. The `goexml-fyne` project parses
this format into Fyne widgets.

## Minimal example

```xml
<goexml version="0.1.0">
  <window title="Hello" size="800x600">
    <vbox>
      <label text="Hello" />
      <button text="Click" onClick="OnClick" />
      <entry text="" placeholder="Name" />
    </vbox>
  </window>
</goexml>
```

## Tags

- `goexml`: root, requires `version="0.1.0"`.
- `window`: one per file, `title`, `size`.
- `vbox`, `hbox`: vertical and horizontal containers.
- `border`: uses `top`, `bottom`, `left`, `right`, `center` children.
- `label`: `text`.
- `button`: `text`, `onClick`.
- `entry`: `text`, `placeholder`.
- `select`: `onChange` required, only `<option>` children, like HTML.
- `option`: `value` and `text` required. The chosen marker is a bare
  `selected=""` (present with no value) on one option; a `selected`
  carrying any value is rejected. The UI shows `text`; `value` is kept
  for the value-reading API.
- `table`: optional `onSelect`, only `<row>` children. The first row is
  the header.
- `row`: only `<cell>` children; every row must have the same number of
  cells.
- `cell`: `text` or inner text — exactly one of the two.
- `text`: long-form text block, `v` or inner text — exactly one of the
  two. `wrap` accepts `word` (default), `break`, or `off`.
- `rect`: a color box `color="#rrggbb"`, for lines and divider bands so
  padding, margin, and alignment are visible to the eye.

## Attributes on every widget

- `id`: unique per file, e.g. `id="btnSave"`. Letters, digits, and
  underscore only; anything else is rejected. Used by handlers, tests,
  and error messages.
- `visible`: `true` or `false`, default `true`.
- `disabled`: only on `button`, `entry`, and `select` (the only widgets
  Fyne can disable); `true` or `false`, default `false`.

## Layout attributes

- `padding`: e.g. `padding="8"`.
- `spacing`: the gap between children in `vbox` and `hbox`, e.g.
  `spacing="4"`.
- `align`: `start`, `center`, or `end`.
- Space filling goes through `style` only — `w-full`, `h-full`, `h-N`.

## Example with ids and layout

```xml
<goexml version="0.1.0">
  <window title="Hello" size="800x600">
    <vbox padding="8" spacing="4" align="start">
      <label id="lblTitle" text="Hello" visible="true" />
      <button id="btnClick" text="Click" onClick="OnClick" />
    </vbox>
  </window>
</goexml>
```

## Recommended form example

```xml
<goexml version="0.1.0">
  <window title="Form" size="800x600">
    <vbox padding="8" spacing="4" align="start">
      <label id="lblTitle" text="Enter name" />
      <entry id="inpName" text="" placeholder="Name" style="w-full" />
      <hbox spacing="4" align="end">
        <button id="btnCancel" text="Cancel" onClick="OnCancel" />
        <button id="btnSave" text="Save" onClick="OnSave" />
      </hbox>
    </vbox>
  </window>
</goexml>
```

## Select and table example

```xml
<vbox id="data" spacing="4" align="start">
  <select id="pickCity" onChange="OnPick">
    <option value="jkt" text="Jakarta" />
    <option value="bdg" text="Bandung" selected="" />
  </select>
  <table id="userTable" onSelect="OnPickRow">
    <row><cell text="Name" /><cell text="Age" /></row>
    <row><cell text="Ann" /><cell text="30" /></row>
    <row><cell text="Bo" /><cell text="25" /></row>
  </table>
</vbox>
```

The first row is always the header. There are no comma-separated
strings anymore — commas inside text are free to use.

Read values back in Go via `win.Find(id)` and the `select` choice via
`win.SelectValue(id)`.

## Grid

A `vbox` with `column="grid"` lays children out in a responsive grid,
not a vertical stack. The column count follows the available width (as
many children as fit at the widest child's width), clamped to
`[min, max]`. Cells are equally wide; the row height follows the
tallest child; leftover vertical space is split evenly across rows.
Consequence: in a grid the final cell height is the row height — `h-N`
on a child only raises the row minimum, it never disables the stretch
(a grid is even; use a vbox when each row needs an exact height).

```xml
<vbox id="gallery" column="grid" min="2" max="4">
  <label id="a" text="A" />
  <label id="b" text="B" />
  <label id="c" text="C" />
</vbox>
```

- `min` / `max` are optional (defaults: min 1, max unbounded apart from
  the child count). Both are integers >= 1, and `min` must not exceed
  `max`.
- `min` / `max` without `column="grid"` is rejected. Any `column` value
  other than `grid` is rejected.
- Only a `vbox` may be a grid. `align` has no effect in grid mode.

## Justify

`justify` controls how the leftover space on the main axis is split in
`vbox` and `hbox`, like Tailwind's `justify-content`. Children carrying
`w-full`/`h-full` are laid out first; `justify` only splits the rest
(if filling children already consumed the leftover, `justify` has no
effect).

- `start` (default): packed at the beginning. `end`: packed at the end.
  `center`: packed in the middle.
- `space-between`: first child at the start, last at the end, the rest
  split evenly across the middle gaps. One child = `start`.
- `space-around`: every child gets equal space on both sides (half at
  the edges). One child = center.
- `space-evenly`: all gaps equal, edges included. One child = center.
- `spacing` stays the minimum gap between children; `justify` only
  splits the remainder.
- `justify` is ignored in grid mode.

```xml
<hbox id="actions" justify="space-between">
  <button id="btnCancel" text="Cancel" onClick="OnCancel" />
  <button id="btnSave" text="Save" onClick="OnSave" />
</hbox>
```

## Style

The `style` attribute works on every widget, space separated, in the
spirit of Tailwind:

- `w-full`: fill the parent width. `h-full`: fill the parent height.
  Both resolve against the **parent**, not siblings.
- In a `vbox`, `w-full` takes the full width without adding height,
  while `h-full` shares the leftover height. In an `hbox` it is the
  other way around: `h-full` takes the full height without adding
  width, while `w-full` shares the leftover width.
- Filling both axes at once: combine them, e.g.
  `style="w-full h-full"` — the child takes the parent's **content
  area**: the parent's padding is never overlapped. (The `expand`
  attribute from early drafts was removed; `style` is the single door.)
- A tag carries at most **one** `w-*` class and **one** `h-*` class.
  More than one — including the `h-full` plus `h-N` mix — is rejected
  with `style has more than one w-*/h-*`.
- `h-N`: an exact height of N*4 px, e.g. `h-2` means 8 px whatever the
  widget's content. `h-0` is a real zero height: an invisible element
  that still takes part in layout through its margin — a spacer.
- Inner padding: `p-N`, `px-N`, `py-N`, `pt-N`, `pr-N`, `pb-N`, `pl-N`.
- Outer margin: `m-N`, `mx-N`, `my-N`, `mt-N`, `mr-N`, `mb-N`, `ml-N`.
- N runs 0-4, each step 4 px.
- Specific wins regardless of order: `p-2 pt-4` means 16 px on top and
  8 px on the other sides.
- Margin sits outside padding: `style="m-2 p-2"` means 8 px outside
  plus 8 px inside.

Example:

```xml
<label id="title" text="Hello" style="w-full p-2" />
```

Unknown `style` utilities are rejected by the parser. N outside 0-4 is
rejected.

## Rules

- One file, one `window`.
- Every widget has a unique `id` of letters, digits, and underscore.
- Tags or attributes outside the lists are rejected by the parser.
- `onClick` is only a name; the function body stays in Go.
- UTF-8 encoding; self-closing tags are allowed for leaves.
