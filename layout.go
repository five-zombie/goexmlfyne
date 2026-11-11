package goexml

import "fyne.io/fyne/v2"

// boxLayout stacks children in one direction with explicit padding,
// spacing, cross-axis alignment and per-child expansion. It ignores
// theme padding so goexml documents render the same on every platform.
// childHint carries one child's sizing wishes into the layout.
type childHint struct {
	wFull    bool    // fill the parent width
	hFull    bool    // fill the parent height
	fixedH   float32 // exact height in px
	fixedSet bool    // an exact height is present, even 0 (h-0)
	wrap     bool    // re-measure after wrapping at the assigned width
}

// tableLayout gives a table its measured size: Fyne's Table.MinSize
// ignores SetColumnWidth, so without this the table would be squeezed
// to its template size and truncate real content.
type tableLayout struct {
	min fyne.Size
}

// newTableLayout returns a layout with a fixed minimum size that its
// single child fills.
func newTableLayout(min fyne.Size) fyne.Layout {
	return &tableLayout{min: min}
}

// MinSize returns the measured content size, never the template size.
func (l *tableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return l.min
}

// Layout fills the child to the container size, never smaller than min.
func (l *tableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	w := max(size.Width, l.min.Width)
	h := max(size.Height, l.min.Height)
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(w, h))
	}
}

type boxLayout struct {
	vertical bool
	padding  float32
	spacing  float32
	align    string // start, center or end on the cross axis
	justify  string // start, center, end, space-between, space-around, space-evenly on the main axis
	hints    []childHint
}

// newBoxLayout returns a vertical (vbox) or horizontal (hbox) layout.
func newBoxLayout(vertical bool, padding, spacing float32, align, justify string, hints []childHint) fyne.Layout {
	return &boxLayout{vertical: vertical, padding: padding, spacing: spacing, align: align, justify: justify, hints: hints}
}

// distribute turns leftover main-axis space into a leading offset and a
// uniform between-children gap. spacing is always the minimum gap;
// justify only shares what is left over.
func (b *boxLayout) distribute(n int, extra float32) (lead, gap float32) {
	switch b.justify {
	case "end":
		return extra, b.spacing
	case "center":
		return extra / 2, b.spacing
	case "space-between":
		if n <= 1 {
			return 0, b.spacing
		}
		return 0, b.spacing + extra/float32(n-1)
	case "space-around":
		return extra / float32(2*n), b.spacing + extra/float32(n)
	case "space-evenly":
		u := extra / float32(n+1)
		return u, b.spacing + u
	default: // "start"
		return 0, b.spacing
	}
}

// shown returns the visible children with their hints.
func (b *boxLayout) shown(objects []fyne.CanvasObject) ([]fyne.CanvasObject, []childHint) {
	var vis []fyne.CanvasObject
	var hints []childHint
	for i, o := range objects {
		if !o.Visible() {
			continue
		}
		vis = append(vis, o)
		if i < len(b.hints) {
			hints = append(hints, b.hints[i])
		} else {
			hints = append(hints, childHint{})
		}
	}
	return vis, hints
}

// MinSize returns the smallest size holding all visible children.
func (b *boxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	vis, hs := b.shown(objects)
	var main, cross float32
	for i, o := range vis {
		m := o.MinSize()
		h := m.Height
		if hs[i].fixedSet {
			h = hs[i].fixedH
		}
		if b.vertical {
			main += h
			if m.Width > cross {
				cross = m.Width
			}
		} else {
			main += m.Width
			if h > cross {
				cross = h
			}
		}
	}
	if len(vis) > 1 {
		main += b.spacing * float32(len(vis)-1)
	}
	if b.vertical {
		return fyne.NewSize(cross+2*b.padding, main+2*b.padding)
	}
	return fyne.NewSize(main+2*b.padding, cross+2*b.padding)
}

// Layout positions visible children inside size.
func (b *boxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	vis, hs := b.shown(objects)
	if len(vis) == 0 {
		return
	}
	sizes := make([]fyne.Size, len(vis))
	var total float32
	for i, o := range vis {
		sizes[i] = o.MinSize()
		h := sizes[i].Height
		if hs[i].fixedSet {
			h = hs[i].fixedH
			sizes[i].Height = h
		}
		if b.vertical {
			total += h
		} else {
			total += sizes[i].Width
		}
	}
	total += b.spacing * float32(len(vis)-1)
	var avail, cross float32
	if b.vertical {
		avail = size.Height - 2*b.padding
		cross = size.Width - 2*b.padding
	} else {
		avail = size.Width - 2*b.padding
		cross = size.Height - 2*b.padding
	}
	// Assign widths first so wrapped text measures against its final
	// width, then grow wrapped children to their full wrapped height.
	// A set fixedH always wins over growth.
	// In a horizontal box a wrapping child can never work at its
	// single-line minimum width (Fyne cannot measure wrapped text
	// below about one word), so wrapping children split the width
	// left over by fixed children equally; spacing stays the gutter.
	hWrapW := float32(0)
	if !b.vertical {
		fixed := b.spacing * float32(len(vis)-1)
		nW := 0
		for i := range vis {
			if hs[i].wrap {
				nW++
			} else {
				fixed += sizes[i].Width
			}
		}
		if nW > 0 && avail > fixed {
			hWrapW = (avail - fixed) / float32(nW)
		}
	}
	for i, o := range vis {
		w := sizes[i].Width
		if b.vertical {
			if hs[i].wFull || hs[i].wrap {
				// A wrapping child fills the cross width like a
				// block element, so it measures against its final
				// width and wraps. Stored back so the final pass
				// keeps it.
				w = cross
			}
		} else if hs[i].wrap && hWrapW > sizes[i].Width {
			w = hWrapW
		}
		sizes[i].Width = w
		o.Resize(fyne.NewSize(w, sizes[i].Height))
	}
	for i, o := range vis {
		// A full-width child also re-measures: at its final width a
		// container sees its real content height (grown text rows),
		// while fixed widgets measure the same and are unaffected.
		if (!hs[i].wrap && !hs[i].wFull) || hs[i].fixedSet {
			continue
		}
		o.Refresh()
		if h := o.MinSize().Height; h > sizes[i].Height {
			if b.vertical {
				total += h - sizes[i].Height
			}
			sizes[i].Height = h
		}
	}
	// Widths changed after measuring (the column split), so re-total
	// for horizontal boxes: justify must distribute what is actually
	// left over, not the pre-split minimum.
	if !b.vertical {
		total = b.spacing * float32(len(vis)-1)
		for i := range vis {
			total += sizes[i].Width
		}
	}
	extra := avail - total
	if extra < 0 {
		extra = 0
	}
	grows := 0
	for _, h := range hs {
		// A wrapping child in a horizontal box already takes its
		// split of the width, so it takes no extra share on top.
		if (b.vertical && h.hFull) || (!b.vertical && h.wFull && !h.wrap) {
			grows++
		}
	}
	share := float32(0)
	leftover := extra
	if grows > 0 {
		share = extra / float32(grows)
		leftover = 0
	}
	lead, gap := b.distribute(len(vis), leftover)
	pos := b.padding + lead
	for i, o := range vis {
		w, h := sizes[i].Width, sizes[i].Height
		if b.vertical {
			if hs[i].wFull {
				w = cross
			}
			if hs[i].hFull {
				h += share
			}
		} else {
			if hs[i].hFull {
				h = cross
			}
			if hs[i].wFull && !hs[i].wrap {
				w += share
			}
		}
		var x, y float32
		if b.vertical {
			y = pos
			switch b.align {
			case "center":
				x = b.padding + (cross-w)/2
			case "end":
				x = b.padding + cross - w
			default:
				x = b.padding
			}
			pos += h + gap
		} else {
			x = pos
			switch b.align {
			case "center":
				y = b.padding + (cross-h)/2
			case "end":
				y = b.padding + cross - h
			default:
				y = b.padding
			}
			pos += w + gap
		}
		o.Move(fyne.NewPos(x, y))
		o.Resize(fyne.NewSize(w, h))
	}
}

// gridLayout flows children row by row into a responsive grid.
// Column count follows the available width (fit as many as the widest
// child allows), clamped to [minCols, maxCols]. Cells share the row
// width equally; leftover height is shared equally by all rows.
type gridLayout struct {
	padding float32
	spacing float32
	minCols int
	maxCols int // <=0 means no upper clamp besides child count
	hints   []childHint
}

// newGridLayout returns a responsive grid layout.
func newGridLayout(padding, spacing float32, minCols, maxCols int, hints []childHint) fyne.Layout {
	if minCols < 1 {
		minCols = 1
	}
	return &gridLayout{padding: padding, spacing: spacing, minCols: minCols, maxCols: maxCols, hints: hints}
}

// columns returns the visible children chunked into rows for the given
// content width, plus the per-cell hints and the cell width.
func (g *gridLayout) columns(objects []fyne.CanvasObject, avail float32) (rows [][]fyne.CanvasObject, rhints [][]childHint, cellW float32) {
	var vis []fyne.CanvasObject
	var hs []childHint
	cellMin := float32(0)
	for i, o := range objects {
		if !o.Visible() {
			continue
		}
		vis = append(vis, o)
		if i < len(g.hints) {
			hs = append(hs, g.hints[i])
		} else {
			hs = append(hs, childHint{})
		}
		if m := o.MinSize().Width; m > cellMin {
			cellMin = m
		}
	}
	n := len(vis)
	if n == 0 {
		return nil, nil, 0
	}
	cols := n
	if cellMin > 0 {
		cols = int((avail + g.spacing) / (cellMin + g.spacing))
	}
	if cols < 1 {
		cols = 1
	}
	if cols < g.minCols {
		cols = g.minCols
	}
	if g.maxCols > 0 && cols > g.maxCols {
		cols = g.maxCols
	}
	if cols > n {
		cols = n
	}
	cellW = avail
	if cols > 0 {
		cellW = (avail - float32(cols-1)*g.spacing) / float32(cols)
		if cellW < 0 {
			cellW = 0
		}
	}
	for i := 0; i < n; i += cols {
		end := i + cols
		if end > n {
			end = n
		}
		rows = append(rows, vis[i:end])
		rhints = append(rhints, hs[i:end])
	}
	return rows, rhints, cellW
}

// MinSize assumes the minimum column count so parents reserve enough room.
func (g *gridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	cols := g.minCols
	if cols < 1 {
		cols = 1
	}
	var vis []fyne.CanvasObject
	for _, o := range objects {
		if o.Visible() {
			vis = append(vis, o)
		}
	}
	n := len(vis)
	if n == 0 {
		return fyne.NewSize(2*g.padding, 2*g.padding)
	}
	if cols > n {
		cols = n
	}
	var wide, total float32
	for r := 0; r < n; r += cols {
		end := r + cols
		if end > n {
			end = n
		}
		var rowH float32
		for _, o := range vis[r:end] {
			m := o.MinSize()
			if m.Width > wide {
				wide = m.Width
			}
			if m.Height > rowH {
				rowH = m.Height
			}
		}
		total += rowH
	}
	rows := (n + cols - 1) / cols
	if rows > 1 {
		total += float32(rows-1) * g.spacing
	}
	return fyne.NewSize(wide*float32(cols)+float32(cols-1)*g.spacing+2*g.padding, total+2*g.padding)
}

// Layout positions visible children in the responsive grid.
func (g *gridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	rows, rhints, cellW := g.columns(objects, size.Width-2*g.padding)
	if len(rows) == 0 {
		return
	}
	var total float32
	rowH := make([]float32, len(rows))
	for r, row := range rows {
		for c, o := range row {
			h := o.MinSize().Height
			if rhints[r][c].fixedSet {
				h = rhints[r][c].fixedH
			}
			if h > rowH[r] {
				rowH[r] = h
			}
			o.Resize(fyne.NewSize(cellW, h))
		}
		total += rowH[r]
	}
	// Grow wrapped text to its full wrapped height, like boxLayout.
	for r, row := range rows {
		for c, o := range row {
			if !rhints[r][c].wrap || rhints[r][c].fixedSet {
				continue
			}
			o.Refresh()
			if h := o.MinSize().Height; h > rowH[r] {
				total += h - rowH[r]
				rowH[r] = h
			}
		}
	}
	if len(rows) > 1 {
		total += float32(len(rows)-1) * g.spacing
	}
	extra := size.Height - 2*g.padding - total
	share := float32(0)
	if extra > 0 {
		share = extra / float32(len(rows))
	}
	y := g.padding
	for r, row := range rows {
		h := rowH[r] + share
		x := g.padding
		for _, o := range row {
			o.Move(fyne.NewPos(x, y))
			o.Resize(fyne.NewSize(cellW, h))
			x += cellW + g.spacing
		}
		y += h + g.spacing
	}
}
