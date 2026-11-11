package goexml

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Window is a decoded goexml document rendered into Fyne objects.
type Window struct {
	// Title is the window title from the XML.
	Title string
	// Size is the window size from the WxH attribute.
	Size fyne.Size
	// Content is the root Fyne object to show.
	Content fyne.CanvasObject
}

// Build renders a validated goexml root node into a Fyne window.
// Every onClick name in the document must have a handler in tap.
func Build(root *Node, tap map[string]func()) (*Window, error) {
	if root == nil || root.Type != "goexml" {
		return nil, fmt.Errorf("goexml: root must be a <goexml> node")
	}
	if len(root.Children) != 1 || root.Children[0].Type != "window" {
		return nil, fmt.Errorf("goexml: line %d: <goexml> must hold one <window>", root.Line)
	}
	w := root.Children[0]
	if len(w.Children) == 0 {
		return nil, fmt.Errorf("goexml: line %d: <window> needs content", w.Line)
	}
	sw, sh, err := parseSize(w.Attr("size"), w.Line)
	if err != nil {
		return nil, err
	}
	var objs []fyne.CanvasObject
	for _, c := range w.Children {
		o, err := buildObject(c, tap)
		if err != nil {
			return nil, err
		}
		objs = append(objs, o)
	}
	var content fyne.CanvasObject
	if len(objs) == 1 {
		content = objs[0]
	} else {
		content = container.New(
			newBoxLayout(true, 0, 4, "start", make([]bool, len(objs))), objs...,
		)
	}
	return &Window{
		Title:   w.Attr("title"),
		Size:    fyne.NewSize(float32(sw), float32(sh)),
		Content: content,
	}, nil
}

// buildObject renders one widget node into a Fyne object.
func buildObject(n *Node, tap map[string]func()) (fyne.CanvasObject, error) {
	switch n.Type {
	case "vbox", "hbox":
		var objs []fyne.CanvasObject
		var expand []bool
		for _, c := range n.Children {
			o, err := buildObject(c, tap)
			if err != nil {
				return nil, err
			}
			objs = append(objs, o)
			expand = append(expand, c.Attr("expand") == "true")
		}
		box := container.New(
			newBoxLayout(n.Type == "vbox", num(n.Attr("padding")), num(n.Attr("spacing")), alignOf(n), expand),
			objs...,
		)
		return applyState(box, n), nil
	case "border":
		var top, bottom, left, right, center fyne.CanvasObject
		for _, r := range n.Children {
			var objs []fyne.CanvasObject
			for _, g := range r.Children {
				o, err := buildObject(g, tap)
				if err != nil {
					return nil, err
				}
				objs = append(objs, o)
			}
			var o fyne.CanvasObject
			switch len(objs) {
			case 0:
				o = nil
			case 1:
				o = objs[0]
			default:
				o = container.New(newBoxLayout(true, 0, 4, "start", make([]bool, len(objs))), objs...)
			}
			switch r.Type {
			case "top":
				top = o
			case "bottom":
				bottom = o
			case "left":
				left = o
			case "right":
				right = o
			case "center":
				center = o
			}
		}
		var centers []fyne.CanvasObject
		if center != nil {
			centers = append(centers, center)
		}
		return applyState(container.NewBorder(top, bottom, left, right, centers...), n), nil
	case "label":
		return applyState(widget.NewLabel(n.Attr("text")), n), nil
	case "button":
		name := n.Attr("onClick")
		fn, ok := tap[name]
		if !ok {
			return nil, fmt.Errorf("goexml: line %d: no handler for onClick %q", n.Line, name)
		}
		return applyState(widget.NewButton(n.Attr("text"), fn), n), nil
	case "entry":
		e := widget.NewEntry()
		if text := n.Attr("text"); text != "" {
			e.SetText(text)
		}
		e.PlaceHolder = n.Attr("placeholder")
		return applyState(e, n), nil
	default:
		return nil, fmt.Errorf("goexml: line %d: cannot build <%s>", n.Line, n.Type)
	}
}

// applyState enforces the visible and disabled attributes.
func applyState(o fyne.CanvasObject, n *Node) fyne.CanvasObject {
	if n.Attr("visible") == "false" {
		o.Hide()
	}
	if n.Attr("disabled") == "true" {
		if d, ok := o.(fyne.Disableable); ok {
			d.Disable()
		}
	}
	return o
}

// alignOf returns the cross-axis alignment, defaulting to start.
func alignOf(n *Node) string {
	if a := n.Attr("align"); a != "" {
		return a
	}
	return "start"
}

// num parses a non-negative integer attribute, defaulting to zero.
func num(s string) float32 {
	i, _ := strconv.Atoi(s)
	return float32(i)
}
