package styles

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// paddingRules maps p-family prefixes to side masks, specific first.
var paddingRules = []struct {
	prefix string
	mask   sideMask
}{
	{"px-", sideX}, {"py-", sideY},
	{"pt-", sideTop}, {"pr-", sideRight}, {"pb-", sideBottom}, {"pl-", sideLeft},
	{"p-", sideAll},
}

// parsePaddingAssign resolves one p-family token into a tiered assignment:
// p-N all sides, px-/py- one axis, pt-/pr-/pb-/pl- one side, N 0-4.
func parsePaddingAssign(dst *Edges, tok string) (tier int, apply func(), handled bool, err error) {
	for _, r := range paddingRules {
		if !strings.HasPrefix(tok, r.prefix) {
			continue
		}
		v, err := strconv.Atoi(tok[len(r.prefix):])
		if err != nil || v < 0 || v > 4 {
			return 0, nil, true, fmt.Errorf("bad padding %q, want %s0 to %s4", tok, r.prefix, r.prefix)
		}
		px, mask := float32(v)*4, r.mask
		return spacingTier(mask), func() { dst.set(mask, px) }, true, nil
	}
	return 0, nil, false, nil
}

// padLayout adds per-side outer padding around a single child.
type padLayout struct {
	pad Edges
}

// NewPad wraps obj in a container with the given padding on every side.
func NewPad(pad Edges, obj fyne.CanvasObject) fyne.CanvasObject {
	return container.New(&padLayout{pad: pad}, obj)
}

// MinSize returns the child size plus padding on every side.
func (p *padLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(p.pad.Left+p.pad.Right, p.pad.Top+p.pad.Bottom)
	}
	m := objects[0].MinSize()
	return fyne.NewSize(m.Width+p.pad.Left+p.pad.Right, m.Height+p.pad.Top+p.pad.Bottom)
}

// Layout centers the single child inside the padding.
func (p *padLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(p.pad.Left, p.pad.Top))
	objects[0].Resize(fyne.NewSize(size.Width-p.pad.Left-p.pad.Right, size.Height-p.pad.Top-p.pad.Bottom))
}
