package styles

// Edges holds per-side spacing in pixels.
type Edges struct {
	Top, Right, Bottom, Left float32
}

// UniformEdges returns equal spacing on all sides.
func UniformEdges(v float32) Edges {
	return Edges{v, v, v, v}
}

// IsZero reports whether all sides are zero.
func (e Edges) IsZero() bool {
	return e == Edges{}
}

type sideMask uint8

const (
	sideTop sideMask = 1 << iota
	sideRight
	sideBottom
	sideLeft
)

const (
	sideAll = sideTop | sideRight | sideBottom | sideLeft
	sideX   = sideLeft | sideRight
	sideY   = sideTop | sideBottom
)

// spacingTier ranks token specificity so side beats axis beats all,
// regardless of token order: p-2 pt-4 and pt-4 p-2 both give top 16 px.
func spacingTier(mask sideMask) int {
	switch mask {
	case sideAll:
		return 0
	case sideX, sideY:
		return 1
	default:
		return 2
	}
}

// set writes v to every side in the mask.
func (e *Edges) set(mask sideMask, v float32) {
	if mask&sideTop != 0 {
		e.Top = v
	}
	if mask&sideRight != 0 {
		e.Right = v
	}
	if mask&sideBottom != 0 {
		e.Bottom = v
	}
	if mask&sideLeft != 0 {
		e.Left = v
	}
}
