# Codegen

Responsibility: turning example XML documents into Go node trees at
build time — the generator, the generated package, and the gates that
keep them in sync. For reading XML at runtime, see
[runtime.md](runtime.md).

## What it is

`cmd/goexmlgen` walks a directory of goexml documents and emits one Go
function per file into a package. The generated tree is the exact
`*Node` graph `Decode` would return, so an app can embed its UI
without shipping XML files or parsing at startup.

The directive lives in `gen/generate.go`:

```go
//go:generate go run ../cmd/goexmlgen -out . -pkg gen ../examples
```

## Regenerate

```sh
go generate ./...
git diff --exit-code   # the tree must be clean afterwards
```

`go generate` rewrites `gen/goexmlgen.go`. A dirty diff after
regeneration means the checked-in tree no longer matches the examples
— CI fails on it, so regenerate and commit together with any XML
change.

## The generated package

Each example file becomes a function named after its stem in
CamelCase: `examples/form.xml` → `gen.Form()`, `examples/page-body.xml`
→ `gen.PageBody()`, and so on. A `goexml-item` fragment is generated
with the same rules as a document.

```go
root := gen.Form()
win, err := goexml.Build(root, handlers, nil, nil)
```

The trees carry source line numbers, so decode/build errors against a
generated tree still point at the original XML line.

## Keeping the tests in sync

Two contracts must hold when examples change:

1. **The sources list.** `gen/gen_test.go` maps every generated
   function to the XML file it came from; a new example needs an
   entry there, or the test fails with a mismatch.
2. **The build contract.** `gen_test.go` builds every generated tree
   the same way an app would:

   - documents build with the shared handler map;
   - `page.xml` gets its container contents;
   - `showcase.xml` gets the demo's resource set, because it carries
     an `<image src="assets/demo.png">`;
   - fragments are injected through `BuildItem` or drafted into a
     `<vbox-expand>` and committed, mirroring the real flows.

## CI gates

The full gate set (also the local pre-commit list):

```sh
gofmt -l .          # formatting
go vet ./...        # static analysis
go build ./...      # compilation
go test -count=1 ./...
go test -race ./...
go generate ./... && git diff --exit-code   # generated tree current
```

The README compile gate (`go test -run TestREADMEExamplesCompile`)
builds every complete Go program in `README.md` against the local
module; it is skipped under `-short` and run explicitly in CI because
a cold dependency cache makes it slow.
