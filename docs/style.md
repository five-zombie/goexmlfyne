# Style

Responsibility: the `style` sub-language — dimension, spacing and
visual utilities, their ranges and their validation rules. Which
elements accept `style` is in [elements.md](elements.md); how the
utilities interact with layout is in [layout.md](layout.md).

The value is a space-separated token list parsed by `styles.ParseStyle`.
Unknown tokens and out-of-range values are decode errors. One px in a
token is one Fyne coordinate unit.

## Dimension utilities

| Token | Effect | Range |
|-------|--------|-------|
| `w-full` | fill the parent width | — |
| `h-full` | fill the parent height | — |
| `h-N` | exact height, `N*4` px | `h-0` … `h-24` |

- At most one `w-*` and one `h-*` per element (mixing `h-full` with
  `h-N` is also rejected).
- `h-0` is a real zero height: an invisible element that still takes
  part in layout through its margin.
- A `table` rejects every `h-*` utility — a table always sizes to its
  content.

## Spacing utilities

| Family | Sides | Range |
|--------|-------|-------|
| `p-N` / `m-N` | all sides | 0–4 (`N*4` px) |
| `px-N`, `py-N` | left+right / top+bottom | 0–4 |
| `pt-N`, `pr-N`, `pb-N`, `pl-N` | one padding side | 0–4 |
| `mt-N`, `mr-N`, `mb-N`, `ml-N` | one margin side | 0–4 |

Specificity is tiered and **order-independent**: a side beats an axis
beats all sides. `p-2 pt-4` and `pt-4 p-2` both give top 16 px.

- Padding wraps the element's content (inner spacing).
- Margin sits outside the element; a hidden element produces a hidden
  margin wrapper so parent layouts collapse the gap.

## Visual utilities

The visual layer (background, border, rounded corners) is drawn
**behind** the element's content and covers its padding; it never
changes the content's size or position.

| Token | Effect |
|-------|--------|
| `bg-[#rrggbb]` | background fill; at most one `bg-` |
| `b-N` | border width, all sides, 0–16 px |
| `b-[#rrggbb]` | border color, all sides |
| `bt-N` / `br-N` / `bb-N` / `bl-N` | border width, one side |
| `bt-[#rrggbb]` / … | border color, one side |
| `rounded` | all corners, 4 px |
| `rounded-md` / `rounded-xl` | all corners, 8 px / 16 px |
| `rounded-N` | all corners, free radius 1–999 px |
| `rounded-{t,b,l,r}-S` | two corners of one side |
| `rounded-{tl,tr,bl,br}-S` | one corner |

`S` in the targeted forms is `md`, `xl`, or `N` (1–999). Specificity
for radii is tiered like spacing: a corner beats a side beats all
corners, regardless of token order.

### Validation rules (merged result, not single tokens)

- A border side draws only with **both** a width and a color on that
  side; either alone is rejected ("has a width but no color" /
  "has a color but no width"), and a color with a zero width is
  rejected.
- `rounded` without any `bg-` and without any drawn border is rejected:
  rounding has nothing to round.
- `rounded` **with** a border requires a uniform border width and a
  uniform border color across sides (`b-N` + `b-[#…]`), because a
  rounded border is one stroked rectangle. Per-side variation with a
  radius is a decode error; per-side variation without a radius is fine
  (drawn as four thin filled rectangles).
- A hidden element produces a hidden visual wrapper, so parent
  layouts collapse the slot.

## Where each element may use the visual layer

- Every widget element accepts the visual tokens; the layer sits
  behind that element's content.
- A `slot` rejects visual tokens at decode (`slot renders no visual
  layer: drop bg-*, b-* and rounded from its style`) — a slot is a
  hole, only its filled content shows. It takes dimension and spacing
  utilities only (`w-full`, `h-N`, `m-*`, `p-*`).
- On an `image` the layer draws behind the pixels; rounded corners do
  not clip them.
- On a `vbox-expand` the layer applies to the whole stack, exactly
  like on any other widget.

## Examples

```xml
<label id="lblTitle" text="Enter name" />
<entry id="inpName" placeholder="Name" style="w-full" />
<vbox id="card" style="bg-[#1e293b] rounded-xl p-2" spacing="2">…</vbox>
<vbox id="panel" style="bg-[#0f172a] b-2 b-[#475569] rounded-md p-2">…</vbox>
<vbox id="accent" style="b-1 bt-3 b-[#334155] bt-[#f59e0b] p-2">…</vbox>
<image id="hero" src="assets/demo.png" style="w-full h-16 bg-[#0f172a] rounded-md" />
```

`padding` as an attribute no longer exists: documents using it are
rejected with `padding moved to style: use the p-N utilities, e.g. p-2
for 8px`.
