// Command goexmlgen turns valid goexml XML files into Go source that
// constructs goexml.Node trees. Every input file — a document or a
// goexml-item fragment — is decoded and validated with the runtime
// parser, so broken XML fails the generation (and the build) with the
// original file name and line. The generated code is backend-neutral:
// Build stays the only backend-aware step.
//
// Usage:
//
//	goexmlgen [-out dir] [-pkg name] input...
//
// Each input is a file or a directory (directories contribute their
// *.xml files, sorted). One generated bundle is written to
// <out>/goexmlgen.go, with one deterministic function per input file,
// named after the file. Ids must be unique across all input files.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/five-zombie/goexmlfyne"
)

// importPath is the module path the generated code imports.
const importPath = "github.com/five-zombie/goexmlfyne"

// funcInfo is one generated function: the file it came from, its kind,
// and the node tree.
type funcInfo struct {
	name string // Go function name
	src  string // input path as given
	kind string // "document" or "fragment"
	root *goexml.Node
}

func main() {
	out := flag.String("out", "gen", "output directory for the generated bundle")
	pkg := flag.String("pkg", "gen", "package name of the generated bundle")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "goexmlgen: no input files or directories")
		os.Exit(2)
	}
	if err := run(flag.Args(), *out, *pkg); err != nil {
		fmt.Fprintln(os.Stderr, "goexmlgen:", err)
		os.Exit(1)
	}
}

// run decodes every input file and writes the generated bundle.
func run(inputs []string, outDir, pkg string) error {
	files, err := inputFiles(inputs)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .xml files in %v", inputs)
	}
	funcs, err := decodeAll(files)
	if err != nil {
		return err
	}
	src, err := emit(funcs, pkg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "goexmlgen.go"), src, 0o644)
}

// inputFiles expands directories into their sorted .xml files.
func inputFiles(inputs []string) ([]string, error) {
	var files []string
	for _, in := range inputs {
		st, err := os.Stat(in)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			if !strings.HasSuffix(in, ".xml") {
				return nil, fmt.Errorf("%s is not an .xml file", in)
			}
			files = append(files, in)
			continue
		}
		matches, err := filepath.Glob(filepath.Join(in, "*.xml"))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	sort.Strings(files)
	return files, nil
}

// decodeAll decodes every file and enforces the global rules: unique
// function names and globally unique ids.
func decodeAll(files []string) ([]funcInfo, error) {
	var funcs []funcInfo
	ids := map[string]string{} // id -> first file claiming it
	used := map[string]bool{}  // function names
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		// Comments may precede the root, so route by trying the
		// document envelope first and falling back to the fragment —
		// the same parser does all the validation either way.
		root, kind, err := decodeAny(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", f, err)
		}
		name := funcName(f)
		if used[name] {
			return nil, fmt.Errorf("%s: function name %s is already used by another input file", f, name)
		}
		used[name] = true
		if err := checkIDs(f, root, ids); err != nil {
			return nil, err
		}
		funcs = append(funcs, funcInfo{name: name, src: f, kind: kind, root: root})
	}
	return funcs, nil
}

// decodeAny decodes a document or a fragment, reporting which kind.
func decodeAny(data []byte) (*goexml.Node, string, error) {
	root, err := goexml.Decode(data)
	if err == nil {
		return root, "document", nil
	}
	root, ferr := goexml.DecodeItem(data)
	if ferr == nil {
		return root, "fragment", nil
	}
	return nil, "", err // the document error is the better diagnosis
}

// checkIDs walks one tree and rejects ids claimed by another file.
// BuildItem already protects document ids at runtime; this moves the
// check to generation time for the files of one run.
func checkIDs(file string, root *goexml.Node, ids map[string]string) error {
	var walk func(n *goexml.Node) error
	walk = func(n *goexml.Node) error {
		if id := n.Attr("id"); id != "" {
			if prev, dup := ids[id]; dup && prev != file {
				return fmt.Errorf("%s:%d: id %q is already used by %s", file, n.Line, id, prev)
			}
			ids[id] = file
		}
		for _, c := range n.Children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root)
}

// funcName derives a deterministic, exported Go name from a file name:
// the base without .xml, split on non-alphanumeric runs, each segment
// capitalized. page-body.xml -> PageBody.
func funcName(file string) string {
	base := strings.TrimSuffix(filepath.Base(file), ".xml")
	var b strings.Builder
	for _, seg := range strings.FieldsFunc(base, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) {
		if seg == "" {
			continue
		}
		b.WriteString(strings.ToUpper(seg[:1]))
		b.WriteString(seg[1:])
	}
	return b.String()
}

// emit renders the bundle and gofmt-normalizes it, so the generated
// code is always formatting-clean.
func emit(funcs []funcInfo, pkg string) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by goexmlgen; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "// Package %s holds goexml node trees generated from XML source\n", pkg)
	fmt.Fprintf(&b, "// files. The trees are validated at generation time and build\n")
	fmt.Fprintf(&b, "// without any XML parsing at runtime.\n")
	fmt.Fprintf(&b, "package %s\n\n", pkg)
	fmt.Fprintf(&b, "import goexml %q\n\n", importPath)
	for _, f := range funcs {
		fmt.Fprintf(&b, "// %s returns the node tree of %s (%s).\n", f.name, f.src, f.kind)
		fmt.Fprintf(&b, "func %s() *goexml.Node {\n\treturn ", f.name)
		emitNode(&b, f.root, 1)
		b.WriteString("\n}\n\n")
	}
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		// A formatting failure is a generator bug; surface the raw
		// source to make it diagnosable.
		return nil, fmt.Errorf("generated code does not format: %v\n%s", err, b.String())
	}
	return src, nil
}

// emitNode writes one node literal at the given indentation depth.
// Attrs is always emitted: the parser allocates an empty map even for
// attribute-less elements, and generated trees must compare equal to
// decoded ones. Text and Children are omitted when empty, matching the
// parser's zero values.
func emitNode(b *strings.Builder, n *goexml.Node, depth int) {
	pad := strings.Repeat("\t", depth)
	fmt.Fprintf(b, "&goexml.Node{\n")
	fmt.Fprintf(b, "%sType:  %s,\n", pad, strconv.Quote(n.Type))
	fmt.Fprintf(b, "%sLine:  %d,\n", pad, n.Line)
	fmt.Fprintf(b, "%sAttrs: map[string]string{", pad)
	keys := make([]string, 0, len(n.Attrs))
	for k := range n.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%s: %s", strconv.Quote(k), strconv.Quote(n.Attrs[k]))
	}
	b.WriteString("},\n")
	if n.Text != "" {
		fmt.Fprintf(b, "%sText:  %s,\n", pad, strconv.Quote(n.Text))
	}
	if len(n.Children) > 0 {
		fmt.Fprintf(b, "%sChildren: []*goexml.Node{\n", pad)
		for _, c := range n.Children {
			fmt.Fprintf(b, "%s\t", pad)
			emitNode(b, c, depth+1)
			b.WriteString(",\n")
		}
		fmt.Fprintf(b, "%s},\n", pad)
	}
	fmt.Fprintf(b, "%s}", pad)
}
