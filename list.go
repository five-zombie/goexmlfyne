package goexml

import (
	"fmt"
	"sort"

	"fyne.io/fyne/v2"
)

// List fronts a built <vbox-expand>: a stack of items drafted from
// goexml-item fragments. Find returns it for the element's id, like
// Container for a container. Draft builds a fragment into an item
// that is not yet on the screen; the app fills it at its own pace
// (templates, slots, images) and hands it to Commit when it is
// complete. Each item owns a private widget scope, so the same
// fragment can be drafted many times without fragment ids colliding.
type List struct {
	// Container holds the visible object: the scroll, or the box when
	// the element is not scrollable.
	*fyne.Container
	box       *fyne.Container // where items go, in commit order
	tap       map[string]func()
	resources map[string]fyne.Resource // the window's set, shared
	items     []*Item                  // committed, in commit order
	drafts    []*Item                  // built, not yet committed
	hints     []childHint              // one per committed item
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

// Item is one list entry: a built goexml-item fragment with its own
// private id space. Draft returns it before it is on the screen;
// every setter the Window has works here too, scoped to this item;
// handlers resolve against the window's handler map at draft time.
// Commit stacks it; Clear retires it.
type Item struct {
	*widgetScope
	list      *List
	node      *Node // the fragment's single child, for commit-time hints
	obj       fyne.CanvasObject
	committed bool
	retired   bool
}

// Draft builds the goexml-item fragment into an item that is not yet
// on the stack: nothing reaches the screen until Commit. The item
// starts with private id spaces and the window's resource set; fill
// it through the Item's setters, Find (slots via Container.SetContent),
// and SetImageRes, then hand it to Commit when it is complete. A
// fragment that does not decode or holds an unknown image key is an
// error — nothing is drafted half-built.
func (l *List) Draft(item *Node) (*Item, error) {
	if item == nil || item.Type != "goexml-item" {
		return nil, fmt.Errorf("goexml: node must be a <goexml-item> root")
	}
	if len(item.Children) != 1 {
		return nil, nodeErr(item, "must hold exactly one child")
	}
	ctx := &buildCtx{
		tap:          l.tap,
		resources:    l.resources,
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
	it := &Item{
		widgetScope: &widgetScope{
			resources:    l.resources,
			byID:         ctx.reg,
			nodes:        ctx.nodes,
			containers:   ctx.containers,
			templates:    ctx.templates,
			selectValues: ctx.selectValues,
			optionRaw:    ctx.optionRaw,
		},
		list: l,
		node: item.Children[0],
		obj:  obj,
	}
	l.drafts = append(l.drafts, it)
	return it, nil
}

// Commit stacks a drafted item at the end of the list: it becomes
// visible, Count grows by one, and further drafts of the same
// fragment keep their own private id spaces. A committed row is
// complete on screen — every container slot must be filled first
// (a filled slot keeps its size; an empty one is a hole), while a
// slot image without src may still be pending: it keeps its size and
// is audited by UnfilledImages. Committing the same item twice, a
// retired item, or an item drafted by another list is an error.
func (l *List) Commit(it *Item) error {
	switch {
	case it == nil:
		return fmt.Errorf("goexml: nothing to commit")
	case it.retired:
		return fmt.Errorf("goexml: this item was cleared by Clear")
	case it.list != l:
		return fmt.Errorf("goexml: this item was drafted by another list")
	case it.committed:
		return fmt.Errorf("goexml: this item is already committed")
	}
	// Unfilled slots are named deterministically: map iteration order
	// is random, so the ids are sorted and the first one reported -
	// the same discipline decode uses for unknown attributes.
	unfilled := make([]string, 0, len(it.containers))
	for id, c := range it.containers {
		if len(c.slot.Objects) == 0 || c.slot.Objects[0] == nil {
			unfilled = append(unfilled, id)
		}
	}
	if len(unfilled) > 0 {
		sort.Strings(unfilled)
		return fmt.Errorf("goexml: slot %q is not filled: set it before Commit", unfilled[0])
	}
	it.committed = true
	for i, d := range l.drafts {
		if d == it {
			l.drafts = append(l.drafts[:i], l.drafts[i+1:]...)
			break
		}
	}
	l.items = append(l.items, it)
	l.hints = append(l.hints, hintOf(it.node))
	l.box.Objects = append(l.box.Objects, it.obj)
	l.box.Refresh()
	return nil
}

// Clear drops every committed item from the stack and retires every
// draft. Handles of cleared items are retired: calling a setter on
// one is an explicit error, never a silent write into a detached
// widget.
func (l *List) Clear() {
	for _, it := range l.items {
		it.retired = true
	}
	for _, it := range l.drafts {
		it.retired = true
	}
	l.items = nil
	l.drafts = nil
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
