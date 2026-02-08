package ghcli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	require.Len(t, diff.Files, 1)

	f := diff.Files[0]
	assert.Equal(t, "main.go", f.Path)
	assert.Empty(t, f.OldPath, "old path should be empty (no rename)")
	require.Len(t, f.Hunks, 2)

	// First hunk: 1 addition among context lines.
	h1 := f.Hunks[0]
	addCount := 0
	for _, l := range h1.Lines {
		if l.Type == domain.DiffAdd {
			addCount++
			assert.Equal(t, `import "os"`, l.Content)
		}
	}
	assert.Equal(t, 1, addCount, "hunk 1 should have 1 addition")

	// Second hunk starts at old line 10.
	h2 := f.Hunks[1]
	assert.NotEmpty(t, h2.Lines, "hunk 2 should have lines")
}

func TestParseDiffEmpty(t *testing.T) {
	diff := ParseDiff("")
	assert.Empty(t, diff.Files, "empty diff should have 0 files")
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
	require.Len(t, diff.Files, 2)
	assert.Equal(t, "a.go", diff.Files[0].Path)
	assert.Equal(t, "b.go", diff.Files[1].Path)
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
	require.Len(t, diff.Files, 1)
	f := diff.Files[0]
	assert.Equal(t, "new.go", f.Path)
	assert.Equal(t, "old.go", f.OldPath)
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
	require.Len(t, diff.Files, 1)
	lines := diff.Files[0].Hunks[0].Lines

	// First context line should be old=5, new=5.
	assert.Equal(t, 5, lines[0].OldNum, "first context old")
	assert.Equal(t, 5, lines[0].NewNum, "first context new")

	// Addition should be new=6, old=0.
	require.Equal(t, domain.DiffAdd, lines[1].Type)
	assert.Equal(t, 6, lines[1].NewNum, "add line new")
	assert.Equal(t, 0, lines[1].OldNum, "add line old")
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
	assert.Equal(t, 2, realLines, "should have 2 real lines")
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
		got := parseRangeStart(tt.input)
		assert.Equal(t, tt.want, got)
	}
}
