package goexml

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "form.xml")
	if err := os.WriteFile(path, []byte(validDoc), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	root, err := Read(path)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if root.Children[0].Attr("title") != "Form" {
		t.Fatalf("title = %q, want Form", root.Children[0].Attr("title"))
	}
	if _, err := Read(filepath.Join(dir, "missing.xml")); err == nil {
		t.Fatal("expected error for missing file, got nil")
	} else if !strings.Contains(err.Error(), "missing.xml") {
		t.Fatalf("error %q does not name the file", err)
	}
}

// BasePath and Read are safe to use concurrently: the base path is
// stored atomically, so -race sees no warning here.
func TestReadBasePathConcurrent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "form.xml"), []byte(validDoc), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	BasePath(dir)
	defer BasePath("")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if _, err := Read("form.xml"); err != nil {
					t.Errorf("Read returned error: %v", err)
					return
				}
				BasePath(dir)
			}
		}()
	}
	wg.Wait()
}

func TestReadBasePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "form.xml"), []byte(validDoc), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	BasePath(dir)
	defer BasePath("")
	root, err := Read("form.xml")
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if root.Children[0].Attr("title") != "Form" {
		t.Fatalf("title = %q, want Form", root.Children[0].Attr("title"))
	}
}

func TestApplyAct(t *testing.T) {
	root, err := Decode([]byte(validDoc))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	act := BuildAct(map[string]func(){
		"OnCancel": func() {},
		"OnSave":   func() {},
	})
	w, err := root.ApplyAct(act)
	if err != nil {
		t.Fatalf("ApplyAct returned error: %v", err)
	}
	if w.Title != "Form" {
		t.Fatalf("title = %q, want Form", w.Title)
	}
	if _, err := root.ApplyAct(BuildAct(map[string]func(){})); err == nil {
		t.Fatal("expected handler error, got nil")
	}
}
