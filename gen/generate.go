// Package gen holds goexml node trees generated from the example XML
// files by goexmlgen; see goexmlgen.go. Regenerate with go generate.
package gen

//go:generate go run ../cmd/goexmlgen -out . -pkg gen ../examples
