# Examples

Responsibility: the catalog of `examples/` documents and how to run
them through the demo. Element and style details are in
[elements.md](elements.md) and [style.md](style.md); the embedding of
these files as Go trees is in [codegen.md](codegen.md).

## Catalog

| File | Kind | Demonstrates |
|------|------|--------------|
| `form.xml` | document | the minimal app: vbox, labels, entry, buttons |
| `widgets.xml` | document | select (with `selected`), table, handlers |
| `showcase.xml` | document | text blocks, grid columns, `space-between` justify, visual cards, `image`, `vbox-expand` |
| `page.xml` | document | `<container>` shell with a slot for Go injection |
| `page-body.xml` | fragment | the body injected into `page.xml`; `{{placeholder}}` templates |
| `feed-item.xml` | fragment | one row of the showcase's `vbox-expand` feed, drafted many times |

Every document declares `version="0.1.0"` and passes the full schema
validation; the fragments hold exactly one child and no window.

## Running the demo

```sh
go run ./cmd/demo            # examples/showcase.xml
go run ./cmd/demo form.xml   # examples/form.xml
go run ./cmd/demo widgets.xml
go run ./cmd/demo page.xml   # containers, injection, templates
```

What each run does:

- **showcase** — builds with the demo's resource set
  (`assets/demo.png`, generated in code so the binary stays
  self-contained), then drafts four rows of `feed-item.xml` into the
  `vbox-expand`, setting each row's title template before committing.
- **page** — builds the shell with a placeholder label in the slot,
  builds `page-body.xml` through `BuildItem`, and `SetContent`s it in;
  the Greet handler reads the entry and re-renders the greeting
  template.

The demo resolves the `examples/` directory via
`goexml.BasePath("examples")` and wraps the content in a scroll, so
tall documents stay usable in a small window.

## Wiring a new example

1. Add the XML under `examples/`, valid against
   [elements.md](elements.md).
2. If it is a document: add a `sources` entry in `gen/gen_test.go`
   (generated function → file) and, when it needs slot contents or
   image resources, teach the gen test's build loop about them.
3. Run `go generate ./...` and commit the regenerated
   `gen/goexmlgen.go` together with the XML — see
   [codegen.md](codegen.md).
4. Optional: add a `case` in `cmd/demo/main.go` if the example needs a
   custom build flow (as `page.xml` does).
