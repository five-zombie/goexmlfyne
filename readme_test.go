package goexml

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// extractGoBlocks walks README.md line by line with a fence state
// machine — no regular expressions. It returns the content of every
// fenced ```go block, in order, together with its starting line number
// for readable failures.
func extractGoBlocks(t *testing.T, readme string) []struct {
	line int
	body string
} {
	t.Helper()
	lines := strings.Split(readme, "\n")
	var blocks []struct {
		line int
		body string
	}
	inBlock, start := false, 0
	var body []string
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		switch {
		case !inBlock && strings.HasPrefix(trimmed, "```go"):
			inBlock, start, body = true, i+1, nil
		case inBlock && strings.HasPrefix(trimmed, "```"):
			inBlock = false
			blocks = append(blocks, struct {
				line int
				body string
			}{start, strings.Join(body, "\n")})
		case inBlock:
			body = append(body, l)
		}
	}
	if inBlock {
		t.Fatalf("README.md: unterminated go fence starting at line %d", start)
	}
	return blocks
}

// TestREADMEExamplesCompile extracts every fenced go block that is a
// complete program (first line `package main`) and compiles it against
// the local module in a temporary module. Partial snippets are skipped:
// only blocks claiming to be runnable programs carry this contract.
func TestREADMEExamplesCompile(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH")
	}
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	blocks := extractGoBlocks(t, string(readme))

	repo, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	sum, sumErr := os.ReadFile(filepath.Join(repo, "go.sum"))

	programs := 0
	for _, b := range blocks {
		first := b.body
		if i := strings.Index(first, "\n"); i >= 0 {
			first = first[:i]
		}
		if strings.TrimSpace(first) != "package main" {
			continue
		}
		programs++

		dir := t.TempDir()
		mod := "module readmeexample\n\ngo 1.27\n\nrequire (\n" +
			"\tfyne.io/fyne/v2 v2.8.2\n" +
			"\tgithub.com/five-zombie/goexmlfyne v0.0.0\n)\n\n" +
			"replace github.com/five-zombie/goexmlfyne => " + repo + "\n"
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o644); err != nil {
			t.Fatal(err)
		}
		if sumErr == nil {
			if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(b.body+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// The minimal go.mod does not carry the dependency graph's
		// indirect requirements; tidy resolves them from the module
		// cache (populated by the repo's own builds) before we build.
		for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}} {
			cmd := exec.Command("go", args...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("README.md go block at line %d: go %s:\n%s", b.line, strings.Join(args, " "), out)
				break
			}
		}
	}
	if programs == 0 {
		t.Fatal("README.md declares no complete go programs to compile")
	}
}
