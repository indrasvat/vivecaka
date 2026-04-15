package reviewprogress

import (
	"crypto/sha1" //nolint:gosec // deterministic digest, not for security
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/indrasvat/vivecaka/internal/cache"
	"github.com/indrasvat/vivecaka/internal/domain"
)

// Scope controls which files are actionable in incremental review mode.
type Scope string

const (
	ScopeAll         Scope = "all"
	ScopeSinceVisit  Scope = "since_visit"
	ScopeSinceReview Scope = "since_review"
	ScopeUnviewed    Scope = "unviewed"
)

// Cycle advances to the next scope in the standard order.
func (s Scope) Cycle() Scope {
	switch s {
	case ScopeSinceVisit:
		return ScopeSinceReview
	case ScopeSinceReview:
		return ScopeUnviewed
	case ScopeUnviewed:
		return ScopeAll
	default:
		return ScopeSinceVisit
	}
}

// Label returns a compact user-facing label.
func (s Scope) Label() string {
	switch s {
	case ScopeSinceVisit:
		return "Since Visit"
	case ScopeSinceReview:
		return "Since Review"
	case ScopeUnviewed:
		return "Unviewed"
	default:
		return "All"
	}
}

// File describes one file under the current review context.
type File struct {
	Path               string
	Additions          int
	Deletions          int
	Status             string
	PatchDigest        string
	Viewed             bool
	ChangedSinceVisit  bool
	ChangedSinceReview bool
	Actionable         bool
}

// Context is the computed incremental review state for a PR revision.
type Context struct {
	Scope                Scope
	HeadSHA              string
	BaseSHA              string
	LastVisitHeadSHA     string
	LastReviewHeadSHA    string
	ViewedFiles          int
	TotalFiles           int
	SinceVisitFiles      int
	SinceReviewFiles     int
	ActionableFiles      int
	Files                []File
	CurrentDigests       map[string]string
	HasReviewBaseline    bool
	HasVisitBaseline     bool
	NextActionablePath   string
	DegradedDigestSource bool
}

// Build derives a review context from file metadata, current digests, and persisted state.
func Build(detail *domain.PRDetail, digests map[string]string, state cache.PRReviewState, degraded bool) *Context {
	if detail == nil {
		return nil
	}

	scope := Scope(state.ActiveScope)
	if scope == "" {
		scope = ScopeSinceReview
	}
	if digests == nil {
		digests = make(map[string]string, len(detail.Files))
	}

	ctx := &Context{
		Scope:                scope,
		HeadSHA:              detail.Branch.HeadSHA,
		BaseSHA:              detail.Branch.BaseSHA,
		LastVisitHeadSHA:     state.LastVisitHeadSHA,
		LastReviewHeadSHA:    state.LastReviewHeadSHA,
		TotalFiles:           len(detail.Files),
		CurrentDigests:       digests,
		HasReviewBaseline:    len(state.LastReviewFiles) > 0 || state.LastReviewHeadSHA != "",
		HasVisitBaseline:     len(state.LastVisitFiles) > 0 || state.LastVisitHeadSHA != "",
		DegradedDigestSource: degraded,
	}

	ctx.Files = make([]File, 0, len(detail.Files))
	for _, fc := range detail.Files {
		digest := digests[fc.Path]
		if digest == "" {
			digest = FallbackDigest(fc)
			ctx.CurrentDigests[fc.Path] = digest
		}

		viewed := false
		if snap, ok := state.ViewedFiles[fc.Path]; ok && snap.PatchDigest == digest {
			viewed = true
		}

		changedSinceVisit := false
		if len(state.LastVisitFiles) > 0 {
			changedSinceVisit = state.LastVisitFiles[fc.Path] != digest
		}

		changedSinceReview := false
		if len(state.LastReviewFiles) > 0 {
			changedSinceReview = state.LastReviewFiles[fc.Path] != digest
		}

		file := File{
			Path:               fc.Path,
			Additions:          fc.Additions,
			Deletions:          fc.Deletions,
			Status:             fc.Status,
			PatchDigest:        digest,
			Viewed:             viewed,
			ChangedSinceVisit:  changedSinceVisit,
			ChangedSinceReview: changedSinceReview,
		}
		file.Actionable = actionable(file, scope, ctx.HasVisitBaseline, ctx.HasReviewBaseline)
		if file.Viewed {
			ctx.ViewedFiles++
		}
		if file.ChangedSinceVisit {
			ctx.SinceVisitFiles++
		}
		if file.ChangedSinceReview {
			ctx.SinceReviewFiles++
		}
		if file.Actionable {
			ctx.ActionableFiles++
			if ctx.NextActionablePath == "" {
				ctx.NextActionablePath = file.Path
			}
		}
		ctx.Files = append(ctx.Files, file)
	}

	return ctx
}

func actionable(file File, scope Scope, hasVisit, hasReview bool) bool {
	switch scope {
	case ScopeSinceVisit:
		if !hasVisit {
			return !file.Viewed
		}
		return file.ChangedSinceVisit || !file.Viewed
	case ScopeSinceReview:
		if !hasReview {
			return !file.Viewed
		}
		return file.ChangedSinceReview || !file.Viewed
	case ScopeUnviewed:
		return !file.Viewed
	default:
		return true
	}
}

// FallbackDigest computes a stable digest from file metadata when a parsed diff is unavailable.
func FallbackDigest(file domain.FileChange) string {
	raw := fmt.Sprintf("%s|%s|%d|%d", file.Path, file.Status, file.Additions, file.Deletions)
	sum := sha1.Sum([]byte(raw)) //nolint:gosec // deterministic content fingerprint, not security-sensitive
	return hex.EncodeToString(sum[:])
}

// DigestsFromDiff computes stable per-file digests from parsed diff content.
func DigestsFromDiff(diff *domain.Diff) map[string]string {
	if diff == nil {
		return nil
	}
	out := make(map[string]string, len(diff.Files))
	for _, file := range diff.Files {
		var b strings.Builder
		b.WriteString(file.Path)
		b.WriteString("\n")
		b.WriteString(file.OldPath)
		b.WriteString("\n")
		for _, hunk := range file.Hunks {
			b.WriteString(hunk.Header)
			b.WriteString("\n")
			for _, line := range hunk.Lines {
				b.WriteString(string(line.Type))
				b.WriteByte('|')
				b.WriteString(line.Content)
				b.WriteByte('|')
				_, _ = fmt.Fprintf(&b, "%d|%d", line.OldNum, line.NewNum)
				b.WriteString("\n")
			}
		}
		sum := sha1.Sum([]byte(b.String())) //nolint:gosec // deterministic content fingerprint, not security-sensitive
		out[file.Path] = hex.EncodeToString(sum[:])
	}
	return out
}

// SnapshotFromContext captures the current file digests as a persisted snapshot.
func SnapshotFromContext(ctx *Context, now time.Time) (headSHA string, files map[string]string) {
	if ctx == nil {
		return "", nil
	}
	files = make(map[string]string, len(ctx.CurrentDigests))
	for path, digest := range ctx.CurrentDigests {
		files[path] = digest
	}
	return ctx.HeadSHA, files
}

// FindFile returns the file at the given path if present.
func (ctx *Context) FindFile(path string) (File, bool) {
	if ctx == nil {
		return File{}, false
	}
	for _, file := range ctx.Files {
		if file.Path == path {
			return file, true
		}
	}
	return File{}, false
}

// NextActionableAfter returns the next actionable file path after the provided path.
func (ctx *Context) NextActionableAfter(path string) string {
	if ctx == nil || ctx.ActionableFiles == 0 {
		return ""
	}

	start := 0
	if path != "" {
		for i, file := range ctx.Files {
			if file.Path == path {
				start = i + 1
				break
			}
		}
	}

	for i := start; i < len(ctx.Files); i++ {
		if ctx.Files[i].Actionable {
			return ctx.Files[i].Path
		}
	}
	for i := 0; i < start && i < len(ctx.Files); i++ {
		if ctx.Files[i].Actionable {
			return ctx.Files[i].Path
		}
	}
	return ""
}
