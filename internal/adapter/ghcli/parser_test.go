package ghcli

import (
	"testing"

	"github.com/indrasvat/vivecaka/internal/domain"
)

const sampleDiff = `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

 import "fmt"
+import "os"

 func main() {
@@ -10,3 +11,4 @@ func main() {
 	fmt.Println("hello")
+	os.Exit(0)
 }
`

func TestParseDiffBasic(t *testing.T) {
	diff := ParseDiff(sampleDiff)

	if len(diff.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(diff.Files))
	}

	f := diff.Files[0]
	if f.Path != "main.go" {
		t.Errorf("file path = %q, want %q", f.Path, "main.go")
	}
	if f.OldPath != "" {
		t.Errorf("old path = %q, want empty (no rename)", f.OldPath)
	}
	if len(f.Hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(f.Hunks))
	}

	// First hunk: 1 addition among context lines.
	h1 := f.Hunks[0]
	addCount := 0
	for _, l := range h1.Lines {
		if l.Type == domain.DiffAdd {
			addCount++
			if l.Content != `import "os"` {
				t.Errorf("added line content = %q, want %q", l.Content, `import "os"`)
			}
		}
	}
	if addCount != 1 {
		t.Errorf("hunk 1: %d additions, want 1", addCount)
	}

	// Second hunk starts at old line 10.
	h2 := f.Hunks[1]
	if len(h2.Lines) == 0 {
		t.Fatal("hunk 2 has no lines")
	}
}

func TestParseDiffEmpty(t *testing.T) {
	diff := ParseDiff("")
	if len(diff.Files) != 0 {
		t.Errorf("empty diff should have 0 files, got %d", len(diff.Files))
	}
}

func TestParseDiffMultipleFiles(t *testing.T) {
	raw := `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1,3 +1,4 @@
 package a
+// new comment

diff --git a/b.go b/b.go
--- /dev/null
+++ b/b.go
@@ -0,0 +1,3 @@
+package b
+
+func B() {}
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 2 {
		t.Fatalf("got %d files, want 2", len(diff.Files))
	}
	if diff.Files[0].Path != "a.go" {
		t.Errorf("file 0 path = %q, want %q", diff.Files[0].Path, "a.go")
	}
	if diff.Files[1].Path != "b.go" {
		t.Errorf("file 1 path = %q, want %q", diff.Files[1].Path, "b.go")
	}
}

func TestParseDiffRename(t *testing.T) {
	raw := `diff --git a/old.go b/new.go
--- a/old.go
+++ b/new.go
@@ -1,3 +1,3 @@
 package pkg
-func Old() {}
+func New() {}
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(diff.Files))
	}
	f := diff.Files[0]
	if f.Path != "new.go" {
		t.Errorf("path = %q, want %q", f.Path, "new.go")
	}
	if f.OldPath != "old.go" {
		t.Errorf("old path = %q, want %q", f.OldPath, "old.go")
	}
}

func TestParseDiffLineNumbers(t *testing.T) {
	raw := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -5,4 +5,5 @@ func foo() {
 	a := 1
+	b := 2
 	return a
 }
`
	diff := ParseDiff(raw)
	if len(diff.Files) != 1 {
		t.Fatalf("got %d files, want 1", len(diff.Files))
	}
	lines := diff.Files[0].Hunks[0].Lines

	// First context line should be old=5, new=5.
	if lines[0].OldNum != 5 || lines[0].NewNum != 5 {
		t.Errorf("first context: old=%d new=%d, want old=5 new=5", lines[0].OldNum, lines[0].NewNum)
	}

	// Addition should be new=6, old=0.
	if lines[1].Type != domain.DiffAdd {
		t.Fatalf("line 1 type = %v, want DiffAdd", lines[1].Type)
	}
	if lines[1].NewNum != 6 {
		t.Errorf("add line new=%d, want 6", lines[1].NewNum)
	}
	if lines[1].OldNum != 0 {
		t.Errorf("add line old=%d, want 0", lines[1].OldNum)
	}
}

func TestParseDiffNoNewlineMarker(t *testing.T) {
	raw := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,2 @@
-old content
+new content
\ No newline at end of file
`
	diff := ParseDiff(raw)
	lines := diff.Files[0].Hunks[0].Lines

	// Should have 2 lines: delete and add. The "no newline" marker is skipped.
	realLines := 0
	for _, l := range lines {
		if l.Type == domain.DiffAdd || l.Type == domain.DiffDelete {
			realLines++
		}
	}
	if realLines != 2 {
		t.Errorf("got %d real lines, want 2", realLines)
	}
}

func TestParseRangeStart(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"10,5", 10},
		{"10", 10},
		{"1,3", 1},
		{"0,0", 0},
		{"100", 100},
	}
	for _, tt := range tests {
		if got := parseRangeStart(tt.input); got != tt.want {
			t.Errorf("parseRangeStart(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
