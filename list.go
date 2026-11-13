package goexml

import (
	"fmt"

	"fyne.io/fyne/v2"
)

// List fronts a built <vbox-expand>: an append-only vertical stack of
// items. Find returns it for the element's id, like Container for a
// container. Items are built from goexml-item fragments and each owns
// a private widget scope, so the same fragment can be appended many
// times without fragment ids colliding.
type List struct {
	// Container holds the visible object: the scroll, or the box when
	// the element is not scrollable.
	*fyne.Container
	box   *fyne.Container // where items go, in append order
	tap   map[string]func()
	items []*Item
	hints []childHint // one per item, kept in append order
}

// listLayout adapts a frozen box or grid layout to a growing stack:
// boxLayout and gridLayout take their per-child hints at construction,
// but a List learns a new hint only when an item is appended. The
// adapter re-points the inner layout's hints at the List's live slice
// on every measure and layout pass, so freshly appended items pick up
// their style hints (w-full, h-N, wrap) exactly like static children
// of a vbox. Same package, so the unexported hints fields are
// reachable.
type listLayout struct {
	inner fyne.Layout
	list  *List
}

func (l *listLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	l.sync()
	return l.inner.MinSize(objects)
}

func (l *listLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	l.sync()
	l.inner.Layout(objects, size)
}

// sync re-points the inner layout's hints. Slice assignment keeps the
// adapter allocation-free: Append grows the List's slice once.
func (l *listLayout) sync() {
	switch inner := l.inner.(type) {
	case *boxLayout:
		inner.hints = l.list.hints
	case *gridLayout:
		inner.hints = l.list.hints
	}
}

// Item is one appended list entry: a built goexml-item fragment with
// its own private id space. Every setter the Window has works here
// too, scoped to this item; handlers resolve against the window's
// handler map at append time.
type Item struct {
	*widgetScope
	retired bool
}

// Append builds the goexml-item fragment into a new item and stacks it
// at the end. contents fills every container slot in the fragment and
// resources holds the image payloads its <image src> keys resolve
// against, both keyed by id, like Build takes them. A fragment that
// does not decode, an unknown slot key, or an unfilled slot is an
// error — nothing appends half-built.
func (l *List) Append(item *Node, contents map[string]fyne.CanvasObject, resources map[string]fyne.Resource) (*Item, error) {
	if item == nil || item.Type != "goexml-item" {
		return nil, fmt.Errorf("goexml: node must be a <goexml-item> root")
	}
	if len(item.Children) != 1 {
		return nil, nodeErr(item, "must hold exactly one child")
	}
	var itemContainers []string
	var pre func(n *Node) error
	pre = func(n *Node) error {
		if n.Type == "container" && n.Attr("id") != "" {
			itemContainers = append(itemContainers, n.Attr("id"))
		}
		for _, c := range n.Children {
			if err := pre(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := pre(item.Children[0]); err != nil {
		return nil, err
	}
	if resources == nil {
		resources = map[string]fyne.Resource{}
	}
	ctx := &buildCtx{
		tap:          l.tap,
		resources:    resources,
		reg:          map[string]fyne.CanvasObject{},
		nodes:        map[string]*Node{},
		containers:   map[string]*Container{},
		templates:    map[string]string{},
		selectValues: map[string]map[string]string{},
		optionRaw:    map[string][]optPair{},
	}
	registerSelects(item.Children[0], ctx)
	obj, err := buildObject(item.Children[0], ctx)
	if err != nil {
		return nil, err
	}
	for id := range contents {
		if !slicesContain(itemContainers, id) {
			return nil, fmt.Errorf("goexml: no container with id %q in the fragment", id)
		}
	}
	for _, id := range itemContainers {
		content, ok := contents[id]
		if !ok {
			return nil, nodeErr(ctx.containers[id].node, "slot must be filled")
		}
		ctx.containers[id].SetContent(content)
	}
	it := &Item{
		widgetScope: &widgetScope{
			resources:    resources,
			byID:         ctx.reg,
			nodes:        ctx.nodes,
			containers:   ctx.containers,
			templates:    ctx.templates,
			selectValues: ctx.selectValues,
			optionRaw:    ctx.optionRaw,
		},
	}
	l.items = append(l.items, it)
	l.hints = append(l.hints, hintOf(item.Children[0]))
	l.box.Objects = append(l.box.Objects, obj)
	l.box.Refresh()
	return it, nil
}

// Clear drops every item from the stack. Handles of cleared items are
// retired: calling a setter on one is an explicit error, never a
// silent write into a detached widget.
func (l *List) Clear() {
	for _, it := range l.items {
		it.retired = true
	}
	l.items = nil
	l.hints = nil
	l.box.Objects = nil
	l.box.Refresh()
}

// Count returns the number of items currently stacked.
func (l *List) Count() int {
	return len(l.items)
}

// Find returns the built object for an id inside this item, or nil
// when absent. Ids stay private to the item: the window's Find does
// not see them.
func (i *Item) Find(id string) fyne.CanvasObject {
	if i.retired {
		return nil
	}
	return i.byID[id]
}

// guard mirrors the scope setters on Item: a retired item
// rejects every setter explicitly.
func (i *Item) guard() error {
	if i.retired {
		return fmt.Errorf("goexml: this item was cleared by Clear")
	}
	return nil
}

// SetText on an item, like Window.SetText but scoped to it.
func (i *Item) SetText(id, v string) error {
	if err := i.guard(); err != nil {
		return err
	}
	return i.widgetScope.SetText(id, v)
}

// Text on an item, like Window.Text but scoped to it.
func (i *Item) Text(id string) (string, error) {
	if err := i.guard(); err != nil {
		return "", err
	}
	return i.widgetScope.Text(id)
}

// SetValue on an item, like Window.SetValue but scoped to it.
func (i *Item) SetValue(id, v string) error {
	if err := i.guard(); err != nil {
		return err
	}
	return i.widgetScope.SetValue(id, v)
}

// SelectValue on an item, like Window.SelectValue but scoped to it.
func (i *Item) SelectValue(id string) (string, bool) {
	if i.retired {
		return "", false
	}
	return i.widgetScope.SelectValue(id)
}

// SetTextFormat on an item, like Window.SetTextFormat but scoped to it.
func (i *Item) SetTextFormat(id string, vals map[string]string) error {
	if err := i.guard(); err != nil {
		return err
	}
	return i.widgetScope.SetTextFormat(id, vals)
}

// SetValueFormat on an item, like Window.SetValueFormat but scoped to
// it.
func (i *Item) SetValueFormat(id string, vals map[string]string) error {
	if err := i.guard(); err != nil {
		return err
	}
	return i.widgetScope.SetValueFormat(id, vals)
}

// SetImage on an item, like Window.SetImage but resolving against the
// item's own resource set.
func (i *Item) SetImage(id, key string) error {
	if err := i.guard(); err != nil {
		return err
	}
	return i.widgetScope.SetImage(id, key)
}

// SetImageRes on an item, like Window.SetImageRes but scoped to it.
func (i *Item) SetImageRes(id string, res fyne.Resource) error {
	if err := i.guard(); err != nil {
		return err
	}
	return i.widgetScope.SetImageRes(id, res)
}

// UnfilledImages on an item: the slot images this item still misses,
// so an append-time audit mirrors the window's. A cleared item has
// nothing left to audit, like its Find returning nil.
func (i *Item) UnfilledImages() []string {
	if i.retired {
		return nil
	}
	return i.widgetScope.UnfilledImages()
}
