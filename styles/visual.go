package styles

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// This file owns the visual layer utilities: bg-[#rrggbb], the border
// family (b-/bt-/br-/bb- taking a color or a width) and the rounded
// family (named sizes, free pixels, per side, per corner). Parsing
// follows the same discipline as padding and margin: token rules,
// specificity tiers via sortAssigns, and validation that rejects
// anything that would render nothing — accepted-then-dead values do
// not exist.

// Color is a parsed "#rrggbb" color.
type Color struct {
	R, G, B uint8
}

// NRGBA converts to an opaque Fyne-compatible color.
func (c Color) NRGBA() color.NRGBA {
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: 255}
}

// parseColor parses "#rrggbb". The styles package cannot reuse the
// root parser's parseHexColor: the root package imports styles, not
// the other way around, so the contract is duplicated deliberately.
func parseColor(s string) (Color, error) {
	bad := func() (Color, error) {
		return Color{}, fmt.Errorf("color must look like #2e2e2e, got %q", s)
	}
	if len(s) != 7 || s[0] != '#' {
		return bad()
	}
	var c Color
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseUint(s[1+2*i:3+2*i], 16, 8)
		if err != nil {
			return bad()
		}
		switch i {
		case 0:
			c.R = uint8(v)
		case 1:
			c.G = uint8(v)
		case 2:
			c.B = uint8(v)
		}
	}
	return c, nil
}

// Visual is the parsed visual layer of a style: background, border
// and rounded corners, rendered as canvas layers behind the content.
type Visual struct {
	// HasBG records a bg- token; BG is its color.
	HasBG bool
	BG    Color
	// BorderW is the border width per side in pixels. A side draws
	// only when it also carries a color (validate enforces the pair).
	BorderW Edges
	// C is the border color per side; hasC records the sides that
	// received a color token, hasW the sides that received a width.
	C          ColorEdges
	hasW, hasC sideMask
	// Radius is the corner radius per corner in pixels.
	Radius Corners
}

// Active reports whether any visual token appeared. With validation
// in place, an active Visual always renders something.
func (v Visual) Active() bool {
	return v.HasBG || v.hasW != 0 || v.hasC != 0 || v.Radius != Corners{}
}

// validate rejects visual combinations that would render nothing or
// render unpredictably. Called from ParseStyle after all tokens are
// applied, so the checks see the merged result, not single tokens.
func (v Visual) validate() error {
	// A border side draws only with both a width and a color; either
	// alone is an accepted-then-dead token.
	if v.hasW != v.hasC {
		for _, s := range sideNames() {
			w, c := v.hasW&s.m != 0, v.hasC&s.m != 0
			if w && !c {
				return fmt.Errorf("border side %s has a width but no color", s.name)
			}
			if c && !w {
				return fmt.Errorf("border side %s has a color but no width", s.name)
			}
		}
	}
	for _, s := range sideNames() {
		if v.hasC&s.m != 0 {
			if w := v.BorderW.get(s.m); w == 0 {
				return fmt.Errorf("border side %s has a color but zero width", s.name)
			}
		}
	}
	anyR := v.Radius.Max() > 0
	drawn := v.hasC != 0
	if anyR && !v.HasBG && !drawn {
		return fmt.Errorf("rounded has nothing to round: add bg- or a border")
	}
	// Rounded borders come from one stroked rectangle, which can only
	// be uniform; per-side variation with a radius is a decode error.
	if anyR && drawn {
		if v.BorderW.Top != v.BorderW.Right || v.BorderW.Top != v.BorderW.Bottom || v.BorderW.Top != v.BorderW.Left {
			return fmt.Errorf("rounded requires a uniform border width: use b-N without per-side variation")
		}
		if v.C.Top != v.C.Right || v.C.Top != v.C.Bottom || v.C.Top != v.C.Left {
			return fmt.Errorf("rounded requires a uniform border color: use b-[#rrggbb] without per-side variation")
		}
	}
	return nil
}

// sideNames lists the sides with names, for validate's messages.
func sideNames() []struct {
	name string
	m    sideMask
} {
	return []struct {
		name string
		m    sideMask
	}{
		{"top", sideTop}, {"right", sideRight}, {"bottom", sideBottom}, {"left", sideLeft},
	}
}

// get returns one side's value from Edges by mask (single-bit masks).
func (e Edges) get(m sideMask) float32 {
	switch m {
	case sideTop:
		return e.Top
	case sideRight:
		return e.Right
	case sideBottom:
		return e.Bottom
	default:
		return e.Left
	}
}

// ColorEdges holds a border color per side.
type ColorEdges struct {
	Top, Right, Bottom, Left Color
}

// set writes c to every side in the mask.
func (c *ColorEdges) set(m sideMask, v Color) {
	if m&sideTop != 0 {
		c.Top = v
	}
	if m&sideRight != 0 {
		c.Right = v
	}
	if m&sideBottom != 0 {
		c.Bottom = v
	}
	if m&sideLeft != 0 {
		c.Left = v
	}
}

// Corners holds a corner radius per corner in pixels.
type Corners struct{ TL, TR, BR, BL float32 }

type cornerMask uint8

const (
	cornerTL cornerMask = 1 << iota
	cornerTR
	cornerBR
	cornerBL
)

const (
	cornerAll    = cornerTL | cornerTR | cornerBR | cornerBL
	cornerTop    = cornerTL | cornerTR
	cornerBottom = cornerBR | cornerBL
	cornerLeft   = cornerTL | cornerBL
	cornerRight  = cornerTR | cornerBR
)

// radiusTier ranks token specificity so a corner beats a side beats
// all, mirroring spacingTier for padding and margin.
func radiusTier(m cornerMask) int {
	switch m {
	case cornerAll:
		return 0
	case cornerTop, cornerBottom, cornerLeft, cornerRight:
		return 1
	default:
		return 2
	}
}

// set writes v to every corner in the mask.
func (c *Corners) set(m cornerMask, v float32) {
	if m&cornerTL != 0 {
		c.TL = v
	}
	if m&cornerTR != 0 {
		c.TR = v
	}
	if m&cornerBR != 0 {
		c.BR = v
	}
	if m&cornerBL != 0 {
		c.BL = v
	}
}

// Max returns the largest corner radius.
func (c Corners) Max() float32 {
	m := c.TL
	if c.TR > m {
		m = c.TR
	}
	if c.BR > m {
		m = c.BR
	}
	if c.BL > m {
		m = c.BL
	}
	return m
}

// applyTo writes the corner radii onto a Fyne rectangle. Uniform
// radii go to the single CornerRadius field; mixed radii set only the
// per-corner overrides, which keeps unset corners at zero whatever
// rule the renderer prefers.
func (c Corners) applyTo(r *canvas.Rectangle) {
	if c.TL == c.TR && c.TR == c.BR && c.BR == c.BL {
		r.CornerRadius = c.TL
		r.TopLeftCornerRadius = 0
		r.TopRightCornerRadius = 0
		r.BottomRightCornerRadius = 0
		r.BottomLeftCornerRadius = 0
		return
	}
	r.CornerRadius = 0
	r.TopLeftCornerRadius = c.TL
	r.TopRightCornerRadius = c.TR
	r.BottomRightCornerRadius = c.BR
	r.BottomLeftCornerRadius = c.BL
}

// radiusDefault and radiusSizes are the named size table, kept on the
// schema's 4 px grid (decision D2 of box-visual-plan).
const radiusDefault = 4

var radiusSizes = map[string]float32{"md": 8, "xl": 16}

// borderRules maps b-family prefixes to side masks, specific first.
// bg- is dispatched before this table: "b-" alone would otherwise
// swallow it.
var borderRules = []struct {
	prefix string
	mask   sideMask
}{
	{"bt-", sideTop}, {"br-", sideRight}, {"bb-", sideBottom}, {"bl-", sideLeft},
	{"b-", sideAll},
}

// parseBGToken resolves one bg- token. Not tiered: a style carries at
// most one background.
func parseBGToken(v *Visual, tok string) (handled bool, err error) {
	if !strings.HasPrefix(tok, "bg-") {
		return false, nil
	}
	val := tok[len("bg-"):]
	if !strings.HasPrefix(val, "[") || !strings.HasSuffix(val, "]") {
		return true, fmt.Errorf("bad background %q, want bg-[#rrggbb]", tok)
	}
	c, err := parseColor(val[1 : len(val)-1])
	if err != nil {
		return true, fmt.Errorf("bad background %q, want bg-[#rrggbb]", tok)
	}
	if v.HasBG {
		return true, fmt.Errorf("more than one bg- token: %q", tok)
	}
	v.HasBG, v.BG = true, c
	return true, nil
}

// parseBorderToken resolves one b-family token into a tiered
// assignment. The value is a bracketed color or a width 0-16 px
// (decision D1 of box-visual-plan).
func parseBorderToken(v *Visual, tok string) (tier int, apply func(), handled bool, err error) {
	for _, r := range borderRules {
		if !strings.HasPrefix(tok, r.prefix) {
			continue
		}
		val := tok[len(r.prefix):]
		if strings.HasPrefix(val, "[") {
			if !strings.HasSuffix(val, "]") {
				return 0, nil, true, fmt.Errorf("bad border color %q, want %s[#rrggbb]", tok, r.prefix)
			}
			c, err := parseColor(val[1 : len(val)-1])
			if err != nil {
				return 0, nil, true, fmt.Errorf("bad border color %q, want %s[#rrggbb]", tok, r.prefix)
			}
			mask := r.mask
			return spacingTier(mask), func() { v.C.set(mask, c); v.hasC |= mask }, true, nil
		}
		w, err := strconv.Atoi(val)
		if err != nil || w < 0 || w > 16 {
			return 0, nil, true, fmt.Errorf("bad border %q, want %s[#rrggbb] or %s0 to %s16", tok, r.prefix, r.prefix, r.prefix)
		}
		px, mask := float32(w), r.mask
		return spacingTier(mask), func() { v.BorderW.set(mask, px); v.hasW |= mask }, true, nil
	}
	return 0, nil, false, nil
}

// cornerRules maps rounded edge names to corner masks, corners first
// so "tl" wins over "l".
var cornerRules = []struct {
	name string
	mask cornerMask
}{
	{"tl", cornerTL}, {"tr", cornerTR}, {"bl", cornerBL}, {"br", cornerBR},
	{"t", cornerTop}, {"b", cornerBottom}, {"l", cornerLeft}, {"r", cornerRight},
}

// parseRadiusToken resolves one rounded-family token into a tiered
// assignment. Bare "rounded" is the default size; a side or corner
// always carries an explicit size, so no form is a no-op.
func parseRadiusToken(v *Visual, tok string) (tier int, apply func(), handled bool, err error) {
	bad := func() (int, func(), bool, error) {
		return 0, nil, true, fmt.Errorf(
			"bad rounded %q, want rounded, rounded-md, rounded-xl, rounded-N (1-999) or rounded-{t,b,l,r,tl,tr,bl,br}-{md,xl,N}", tok)
	}
	if tok == "rounded" {
		return 0, func() { v.Radius.set(cornerAll, radiusDefault) }, true, nil
	}
	if !strings.HasPrefix(tok, "rounded-") {
		return 0, nil, false, nil
	}
	parts := strings.Split(tok[len("rounded-"):], "-")
	size := func(s string) (float32, bool) {
		if n, ok := radiusSizes[s]; ok {
			return n, true
		}
		x, err := strconv.Atoi(s)
		if err != nil || x < 1 || x > 999 {
			return 0, false
		}
		return float32(x), true
	}
	switch len(parts) {
	case 1:
		s, ok := size(parts[0])
		if !ok {
			return bad()
		}
		return 0, func() { v.Radius.set(cornerAll, s) }, true, nil
	case 2:
		var mask cornerMask
		for _, r := range cornerRules {
			if r.name == parts[0] {
				mask = r.mask
				break
			}
		}
		if mask == 0 {
			return bad()
		}
		s, ok := size(parts[1])
		if !ok {
			return bad()
		}
		return radiusTier(mask), func() { v.Radius.set(mask, s) }, true, nil
	default:
		return bad()
	}
}

// NewVisual builds the visual layer behind obj: a filled rounded
// rectangle for the background, border strokes, and the content on
// top. Border widths are drawn inside the bounds, so the layer never
// changes the content's size or position contract. A hidden obj
// produces a hidden wrapper, like NewMargin, so parent box layouts
// collapse the slot.
func NewVisual(v Visual, obj fyne.CanvasObject) fyne.CanvasObject {
	l := &visualLayout{v: v}
	var layers []fyne.CanvasObject
	anyR := v.Radius.Max() > 0
	if v.HasBG {
		r := canvas.NewRectangle(v.BG.NRGBA())
		v.Radius.applyTo(r)
		l.hasBG = true
		layers = append(layers, r)
	}
	if v.hasC != 0 {
		if anyR {
			// Validation guarantees a uniform border here: one stroked
			// rounded rectangle carries it.
			b := canvas.NewRectangle(color.NRGBA{})
			b.StrokeColor = v.C.Top.NRGBA()
			b.StrokeWidth = v.BorderW.Top
			v.Radius.applyTo(b)
			l.stroke = true
			layers = append(layers, b)
		} else {
			l.edges = true
			// Per-side borders are thin filled rectangles, fully inside
			// the bounds; undrawn sides get zero-size placeholders.
			for _, s := range []struct {
				m sideMask
				c Color
			}{{sideTop, v.C.Top}, {sideRight, v.C.Right}, {sideBottom, v.C.Bottom}, {sideLeft, v.C.Left}} {
				layers = append(layers, canvas.NewRectangle(s.c.NRGBA()))
			}
		}
	}
	layers = append(layers, obj)
	c := container.New(l, layers...)
	if !obj.Visible() {
		c.Hide()
	}
	return c
}

// visualLayout arranges the visual layer: canvas rects fill the whole
// area behind the content, per-side border rects hug their edge, and
// corner radii are clamped to half the smaller dimension so a radius
// can never produce an undefined shape.
type visualLayout struct {
	v      Visual
	hasBG  bool
	stroke bool
	edges  bool
}

func (l *visualLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	// The layer is inset: only the content contributes to min size.
	return objects[len(objects)-1].MinSize()
}

// maxF clamps a computed length to the non-negative range: when the
// border is thicker than the element, the inset stroke rect would
// invert; draw it empty instead of mirroring it.
func maxF(v float32) float32 {
	if v < 0 {
		return 0
	}
	return v
}

func (l *visualLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	w, h := size.Width, size.Height
	clamp := func(r float32) float32 {
		half := min(w, h) / 2
		if r > half {
			return half
		}
		return r
	}
	clamped := Corners{
		clamp(l.v.Radius.TL), clamp(l.v.Radius.TR),
		clamp(l.v.Radius.BR), clamp(l.v.Radius.BL),
	}
	i := 0
	if l.hasBG {
		clamped.applyTo(objects[i].(*canvas.Rectangle))
		objects[i].Move(fyne.NewPos(0, 0))
		objects[i].Resize(size)
		i++
	}
	if l.stroke {
		// A stroked rectangle draws centered on its path: at the plain
		// bounds half the thickness would hang outside the element.
		// Inset by half the width so the stroke's outer edge sits on
		// the bounds — the same geometry the per-side paths have.
		s := l.v.BorderW.Top
		clamped.applyTo(objects[i].(*canvas.Rectangle))
		objects[i].Move(fyne.NewPos(s/2, s/2))
		objects[i].Resize(fyne.NewSize(maxF(w-s), maxF(h-s)))
		i++
	}
	if l.edges {
		edges := []struct {
			m            sideMask
			x, y, ew, eh float32
		}{
			{sideTop, 0, 0, w, l.v.BorderW.Top},
			{sideRight, w - l.v.BorderW.Right, 0, l.v.BorderW.Right, h},
			{sideBottom, 0, h - l.v.BorderW.Bottom, w, l.v.BorderW.Bottom},
			{sideLeft, 0, 0, l.v.BorderW.Left, h},
		}
		for _, e := range edges {
			objects[i].Move(fyne.NewPos(e.x, e.y))
			objects[i].Resize(fyne.NewSize(e.ew, e.eh))
			i++
		}
	}
	content := objects[len(objects)-1]
	content.Move(fyne.NewPos(0, 0))
	content.Resize(size)
}
