package domain

// Diff represents the diff content for a PR.
type Diff struct {
	Files []FileDiff `json:"files"`
}

// FileDiff represents the diff for a single file.
type FileDiff struct {
	Path    string `json:"path"`
	Hunks   []Hunk `json:"hunks"`
	OldPath string `json:"old_path,omitempty"` // for renames
}

// Hunk represents a contiguous block of changes.
type Hunk struct {
	Header string     `json:"header"`
	Lines  []DiffLine `json:"lines"`
}

// DiffLine represents a single line in a diff hunk.
type DiffLine struct {
	Type    DiffLineType `json:"type"`
	Content string       `json:"content"`
	OldNum  int          `json:"old_num,omitempty"`
	NewNum  int          `json:"new_num,omitempty"`
}

// DiffLineType indicates whether a line was added, deleted, or is context.
type DiffLineType string

const (
	DiffAdd     DiffLineType = "add"
	DiffDelete  DiffLineType = "delete"
	DiffContext DiffLineType = "context"
)

func (t DiffLineType) String() string { return string(t) }
