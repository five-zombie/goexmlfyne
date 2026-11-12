package goexml

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"fyne.io/fyne/v2"
)

// basePath is the directory relative Read paths resolve against.
// It is stored atomically, so BasePath is safe to call while other
// goroutines are inside Read.
var basePath atomic.Value

// BasePath sets the directory that Read prepends to relative paths,
// e.g. BasePath("examples") then Read("form.xml"). Absolute paths are
// never rewritten.
func BasePath(dir string) {
	basePath.Store(dir)
}

// Read loads a goexml document from path and decodes it in one call.
// Relative paths resolve against BasePath when one is set.
func Read(path string) (*Node, error) {
	if bp, ok := basePath.Load().(string); ok && bp != "" && !filepath.IsAbs(path) {
		path = filepath.Join(bp, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("goexml: read %s: %w", path, err)
	}
	return Decode(data)
}

// ReadItem loads a goexml-item fragment from path and decodes it in
// one call. Relative paths resolve against BasePath when one is set.
func ReadItem(path string) (*Node, error) {
	if bp, ok := basePath.Load().(string); ok && bp != "" && !filepath.IsAbs(path) {
		path = filepath.Join(bp, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("goexml: read %s: %w", path, err)
	}
	return DecodeItem(data)
}

// Act maps handler names used in onClick, onChange and onSelect
// attributes to Go functions. Build one with BuildAct.
type Act map[string]func()

// BuildAct wraps a handler map so it can be applied to a decoded tree.
func BuildAct(handlers map[string]func()) Act {
	return Act(handlers)
}

// ApplyAct renders the node into a Fyne window using the given handlers.
// Act shares its underlying type with the handler map Build takes.
// Documents with <container> need ApplyActContents instead: Build
// requires content for every slot, so ApplyAct alone cannot render them.
func (n *Node) ApplyAct(act Act) (*Window, error) {
	return Build(n, act, nil)
}

// ApplyActContents renders the node like ApplyAct and additionally
// supplies the slot contents for the document's containers, keyed by
// container id — the same map Build takes.
func (n *Node) ApplyActContents(act Act, contents map[string]fyne.CanvasObject) (*Window, error) {
	return Build(n, act, contents)
}
