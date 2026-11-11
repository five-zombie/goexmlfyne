package goexml

import "fyne.io/fyne/v2"

// boxLayout stacks children in one direction with explicit padding,
// spacing, cross-axis alignment and per-child expansion. It ignores
// theme padding so goexml documents render the same on every platform.
type boxLayout struct {
	vertical bool
	padding  float32
	spacing  float32
	align    string // start, center or end on the cross axis
	expand   []bool // per-child expansion along the main axis
}

// newBoxLayout returns a vertical (vbox) or horizontal (hbox) layout.
func newBoxLayout(vertical bool, padding, spacing float32, align string, expand []bool) fyne.Layout {
	return &boxLayout{vertical: vertical, padding: padding, spacing: spacing, align: align, expand: expand}
}

// shown returns the visible children with their expansion flags.
func (b *boxLayout) shown(objects []fyne.CanvasObject) ([]fyne.CanvasObject, []bool) {
	var vis []fyne.CanvasObject
	var exp []bool
	for i, o := range objects {
		if !o.Visible() {
			continue
		}
		vis = append(vis, o)
		exp = append(exp, i < len(b.expand) && b.expand[i])
	}
	return vis, exp
}

// MinSize returns the smallest size holding all visible children.
func (b *boxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	vis, _ := b.shown(objects)
	var main, cross float32
	for _, o := range vis {
		m := o.MinSize()
		if b.vertical {
			main += m.Height
			if m.Width > cross {
				cross = m.Width
			}
		} else {
			main += m.Width
			if m.Height > cross {
				cross = m.Height
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
	vis, exp := b.shown(objects)
	if len(vis) == 0 {
		return
	}
	sizes := make([]fyne.Size, len(vis))
	var total float32
	for i, o := range vis {
		sizes[i] = o.MinSize()
		if b.vertical {
			total += sizes[i].Height
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
	extra := avail - total
	if extra < 0 {
		extra = 0
	}
	grows := 0
	for _, e := range exp {
		if e {
			grows++
		}
	}
	share := float32(0)
	if grows > 0 {
		share = extra / float32(grows)
	}
	pos := b.padding
	for i, o := range vis {
		w, h := sizes[i].Width, sizes[i].Height
		if b.vertical {
			if exp[i] {
				h += share
				w = cross
			}
		} else if exp[i] {
			w += share
			h = cross
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
			pos += h + b.spacing
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
			pos += w + b.spacing
		}
		o.Move(fyne.NewPos(x, y))
		o.Resize(fyne.NewSize(w, h))
	}
}
