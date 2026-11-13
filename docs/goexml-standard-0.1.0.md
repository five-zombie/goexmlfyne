# The goexml 0.1.0 standard

The `goexml` standard is the XML format. The `goexml-fyne` project parses
this format into Fyne widgets.

> **Amendment, 13 November 2026 (still version 0.1.0):** the `style`
> sub-language gains the **visual layer** — a background (`bg-[#rrggbb]`),
> borders (`b-`), and rounded corners (`rounded`) drawn behind an
> element's content. See the *Visual* block in the Style section. The
> element and attribute lists are unchanged.
>
> **Second amendment, 13 November 2026 (still version 0.1.0):** the leaf
> element **`image`** joins the schema: `src` names a resource key in
> the runtime's registered set and is validated at build when present —
> a key may look like a path (`assets/logo.png`) but nothing is read
> from disk or network; without `src` the image is a runtime-filled
> slot that must be filled before showing. `fit` is `contain`
> (default), `cover` or `fill`. The visual layer applies behind the
> pixels; rounded corners do not clip
> them. See the Tags list.
>
> **Third amendment, 13 November 2026 (still version 0.1.0):** the
> structure element **`vbox-expand`** joins the schema: an append-only
> stack whose items are built at runtime from `goexml-item` fragments.
> The element takes no children. See the Tags list.
>
> **Fourth amendment, 13 November 2026 (still version 0.1.0):** `id`
> becomes **optional** on possibly-static elements. It stays required
> on the elements the runtime must reference by name: `container`,
> `vbox-expand`, an `image` without `src`, and any element carrying a
> `{{placeholder}}` template. Uniqueness and the `[A-Za-z0-9_]+`
> shape still apply whenever an id is present. See the common
> attributes block.
>
> **Fifth amendment, 13 November 2026 (still version 0.1.0):** the
> `padding` attribute is removed from `vbox` and `hbox` — the style
> utilities (`p-2` = 8 px) superseded it; documents using the
> attribute are rejected with a migration hint. In the same
> amendment, `vbox-expand` gains every static box layout feature:
> `spacing`, `align`, `justify`, and `column="grid"` with `min`/`max`
> now lay out its appended items exactly as they lay out a `vbox`'s
> children. See Layout attributes and the Tags list.
>
> **Sixth amendment, 13 November 2026 (still version 0.1.0):** flag
> attributes — attributes whose presence alone means true, currently
> `scrollable` and `selected` — may be written bare: the decoder
> rewrites a bare flag to `="true"` before parsing, so
> `<vbox-expand scrollable />` and `scrollable="true"` spell the same
> thing. The older empty form (`scrollable=""`) and any other value
> are rejected with a migration hint naming the accepted spellings.
>
> **Seventh amendment, 13 November 2026 (still version 0.1.0):** the
> runtime flow for `vbox-expand` becomes draft/commit. `List.Draft`
> builds a fragment into an item that is not yet on the screen; the
> application fills it — text templates, slots through
> `Container.SetContent`, images through `SetImageRes` — and hands it
> to `List.Commit`, which refuses an unfilled slot so a committed row
> is complete on screen. `List.Append(frag, contents, resources)` is
> gone: slot fills and image payloads were per-row data either way,
> and the setters express it without map literals in the loop. A
> fragment's `<image src="key">` resolves against the build's
> resource set.

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
- `window`: one per file, `title`, `size`, exactly one child — wrap
  multiple children in a `vbox` yourself.
- `container`: an injection frame, requires `id`. It holds free design
  (any widget) and **exactly one** `<slot>` somewhere in its subtree.
  Containers never nest: a container must not be a descendant of a
  container within one file. Multiple sibling containers are fine.
- `slot`: the hole filled from Go. A leaf whose `style` takes only
  the dimension utilities — `w-*`, `h-N`, padding and margin — that
  decide the area it claims; the visual utilities (`bg-`, `b-`,
  `rounded`) are rejected on a slot, because a slot is a hole with
  nothing to paint on. `Build` requires content for every container
  id, and `SetContent` on the built `*Container` replaces the slot's
  content. A slot outside any `<container>` is rejected.
- `goexml-item`: the root of an injectable fragment, requires
  `version="0.1.0"` — the fragment envelope. It holds **exactly one**
  child widget (a window is not allowed inside); the same schema rules
  apply, and its ids join the document's global id space. An item is
  only legal as a fragment root, never nested anywhere.
- `vbox`, `hbox`: vertical and horizontal containers.
- `vbox-expand`: a stack of items drafted at runtime from
  `goexml-item` fragments. Takes no children and no
  items in the document; its layout attributes mirror the static
  `vbox` — `spacing`, `align`, `justify`, and `column="grid"` with
  `min`/`max` — applied to the committed items, and the bare
  `scrollable` attribute wraps the stack in a scroll so `h-*` bounds
  its visible height. A third amendment element — see the amendment
  block.
- `border`: uses `top`, `bottom`, `left`, `right`, `center` children;
  each region holds at most one child.
- `label`: `text`.
- `button`: `text`, `onClick`.
- `entry`: `text`, `placeholder`.
- `select`: `onChange` required, only `<option>` children, like HTML.
- `option`: `value` and `text` required. The chosen marker is the
  bare `selected` attribute — the decoder rewrites bare flags to
  `="true"`, so `selected="true"` spells the same thing — on one
  option; any other value is rejected. The UI shows `text`; `value`
  is kept for the value-reading API.
- `table`: optional `onSelect`, only `<row>` children. The first row is
  the header. Height style utilities (`h-full`, `h-N`) are not
  supported on a table: a table always sizes to its content.
- `row`: only `<cell>` children; every row must have the same number of
  cells.
- `cell`: `text` or inner text — exactly one of the two.
- `text`: long-form text block, `v` or inner text — exactly one of the
  two. `wrap` accepts `word` (default), `break`, or `off`.
- `rect`: a color box `color="#rrggbb"`, for lines and divider bands so
  padding, margin, and alignment are visible to the eye.
- `image`: a picture from the runtime's registered resource set.
  `src` optionally names a resource key (letters, digits, dot, dash,
  slash, underscore; a key may look like a path — `assets/logo.png` —
  but nothing is ever read from disk or network, and never a
  template);
  `fit` is `contain` (default), `cover` or `fill`. With `src`, the
  build fails on an unknown key or undecodable bytes — no image ever
  renders as a silent empty box. Without `src`, the image is a
  runtime-filled slot: it renders empty until the runtime fills it
  (like a container slot), and the runtime must fill it before
  showing and can audit leftovers. Without size style a filled image
  is as large as its pixels; the dimension utilities size it like any
  leaf (`h-*` is allowed, unlike on a table), and the visual layer
  draws behind the pixels (rounded corners do not clip them). The
  runtime can fill or swap the resource by id at runtime (see the
  implementation's image setters); unknown keys and undecodable bytes
  are rejected there too.

## Attributes on every widget

- `id`: optional, unique per file when present, e.g. `id="btnSave"`.
  Letters, digits, and underscore only; anything else is rejected.
  Give it to elements the runtime addresses; skip it on possibly-
  static ones. Still required on `container`, `vbox-expand`, an
  `image` without `src`, and any element whose text carries a
  `{{placeholder}}` template — those cannot work unaddressed.
- `visible`: `true` or `false`, default `true`.
- `disabled`: only on `button`, `entry`, and `select` (the only widgets
  Fyne can disable); `true` or `false`, default `false`.

## Layout attributes

- `spacing`: the gap between children in `vbox`, `hbox`, and
  `vbox-expand`, e.g. `spacing="4"`. Inner padding is a style concern:
  use the `p-N` utilities (the old `padding` attribute moved there —
  see the fifth amendment block).
- `align`: `start`, `center`, or `end`.
- `justify`, `column="grid"`, `min`, `max` work on `vbox` as before
  and, since the fifth amendment, on `vbox-expand` — where they lay
  out the committed items.
- Space filling goes through `style` only — `w-full`, `h-full`, `h-N`.

## Example with ids and layout

```xml
<goexml version="0.1.0">
  <window title="Hello" size="800x600">
    <vbox style="p-2" spacing="4" align="start">
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
    <vbox style="p-2" spacing="4" align="start">
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
    <option value="bdg" text="Bandung" selected />
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
rejected (border widths below are the exception: direct pixels).

### Visual

On top of the utilities above, a style can attach a **visual layer**
behind an element's content: a background, borders, and rounded
corners. The layer paints inside the element's bounds — it never moves
or grows the content, and padding stays the inner spacing on top of
it:

- `bg-[#rrggbb]`: a solid background behind the whole element,
  including its padding area. At most one `bg-` token per element.
- `b-[#rrggbb]`: border color for all four sides; `bt-`, `br-`, `bl-`,
  `bb-[#rrggbb]` color one side.
- `b-N`: border width for all four sides, **direct pixels** 0-16
  (`b-1` is a 1 px border — border widths are not the 4 px step grid).
  `bt-N`, `br-N`, `bl-N`, `bb-N` size one side.
- Rounded corners: `rounded` (4 px), `rounded-md` (8 px),
  `rounded-xl` (16 px), `rounded-N` (N px, 1-999); per side
  `rounded-t-*`, `rounded-b-*`, `rounded-l-*`, `rounded-r-*`; per
  corner `rounded-tl-*`, `rounded-tr-*`, `rounded-bl-*`,
  `rounded-br-*`. A side or corner form requires an explicit size.

Rules of the visual layer:

- Every side carrying a border **width** must carry a border **color**,
  and the reverse — `b-2` alone, a color alone, or a mix that leaves
  some sides half-configured is rejected, as is a colored side with
  zero width. A half-configured border must not silently render as
  nothing.
- A border drawn with rounded corners is a single stroked rounded
  rectangle, which can only be uniform: **`rounded` with a border
  requires a uniform border** — `b-N` only, no per-side variation. A
  rounded background with no border, and a per-side border without
  rounding, are both fine.
- A radius larger than half the element is clamped at render to half
  the smaller dimension.
- Border widths and corners draw **inside** the element's bounds: they
  do not change the element's size or its neighbors' positions.
- `rounded` with nothing to round — no background and no border — is
  rejected.

Example:

```xml
<vbox id="card" style="bg-[#0f172a] b-2 b-[#475569] rounded-md p-2">
  <label id="t" text="Card" />
</vbox>
```

## Rules

- One file, one `window`.
- Every widget has a unique `id` of letters, digits, and underscore.
- Tags or attributes outside the lists are rejected by the parser.
  The `style` attribute is its own small language: its utilities are
  listed in the Style section, and an unknown one there is rejected
  the same way.
- Comments are allowed anywhere. Directives and processing
  instructions are only allowed in the prolog; inside an element they
  are rejected.
- `onClick` is only a name; the function body stays in Go.
- UTF-8 encoding; self-closing tags are allowed for leaves.
- Dynamic values: static text stays in attributes; values that change
  at runtime are set from Go by id (`SetText`, `SetValue`, and the
  `{{placeholder}}` template renderers — see the library API). Only
  `{{token}}` with letters, digits, and underscore is a valid
  placeholder; a malformed one is rejected at decode. There is no
  logic in XML.

## Code generation

The `goexmlgen` tool turns valid goexml files into Go source at build
time, so the XML is part of the binary instead of a runtime asset:

- `go generate` runs `cmd/goexmlgen` over the configured input
  directory. Every `.xml` file (documents and `goexml-item` fragments)
  is decoded and validated with the same parser the runtime uses.
- Any XML error — syntax, unknown tag, orphan slot, nested container,
  duplicate id, wrong version — fails the generation, so broken XML
  becomes a compile error on the developer machine, with the original
  file name and line.
- The generated code only constructs `goexml.Node` trees: one
  deterministic function per input file, named after the file. It is
  backend-neutral — `Build` stays the only backend-aware step — and
  each node keeps its original source line for runtime error messages.
- Ids are checked for uniqueness across all input files of one
  generation run: a fragment and its document cannot collide.
- The generated code carries a "generated, do not edit" header. CI
  regenerates and rejects any diff, so generated code is always
  up to date.
- Runtime decoding is unchanged: pages can still come from XML at
  runtime. Generated nodes and runtime-decoded nodes are
  interchangeable inputs to `Build` and `BuildItem`.
