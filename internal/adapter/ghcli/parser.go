package ghcli

import (
	"strings"

	"github.com/indrasvat/vivecaka/internal/domain"
)

// ParseDiff parses a unified diff string into a domain.Diff.
func ParseDiff(raw string) domain.Diff {
	p := &diffParser{}
	return p.parse(raw)
}

// diffParser holds state while parsing a unified diff.
type diffParser struct {
	oldLine int
	newLine int
}

func (p *diffParser) parse(raw string) domain.Diff {
	var diff domain.Diff
	var currentFile *domain.FileDiff
	var currentHunk *domain.Hunk

	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
					currentHunk = nil
				}
				diff.Files = append(diff.Files, *currentFile)
			}
			currentFile = &domain.FileDiff{}
			currentHunk = nil
			parseDiffHeader(line, currentFile)

		case strings.HasPrefix(line, "--- "):
			// Old file path - handled by diff header.

		case strings.HasPrefix(line, "+++ "):
			if currentFile != nil && currentFile.Path == "" {
				path := strings.TrimPrefix(line, "+++ ")
				path = strings.TrimPrefix(path, "b/")
				currentFile.Path = path
			}

		case strings.HasPrefix(line, "@@ "):
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				currentHunk = &domain.Hunk{Header: line}
				p.parseHunkHeader(line)
			}

		case currentHunk != nil:
			dl := p.parseDiffLine(line)
			if dl != nil {
				currentHunk.Lines = append(currentHunk.Lines, *dl)
			}
		}
	}

	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		diff.Files = append(diff.Files, *currentFile)
	}

	return diff
}

// parseDiffHeader extracts file paths from a "diff --git a/path b/path" line.
func parseDiffHeader(line string, f *domain.FileDiff) {
	_, after, ok := strings.Cut(line, " b/")
	if ok {
		f.Path = after
	}
	if idx := strings.Index(line, " a/"); idx != -1 {
		rest := line[idx+3:]
		if before, _, ok := strings.Cut(rest, " b/"); ok && before != f.Path {
			f.OldPath = before
		}
	}
}

// parseHunkHeader extracts starting line numbers from "@@ -old,count +new,count @@".
func (p *diffParser) parseHunkHeader(line string) {
	p.oldLine = 1
	p.newLine = 1

	parts := strings.SplitN(line, "@@", 3)
	if len(parts) < 2 {
		return
	}
	for _, r := range strings.Fields(strings.TrimSpace(parts[1])) {
		if strings.HasPrefix(r, "-") {
			if n := parseRangeStart(r[1:]); n > 0 {
				p.oldLine = n
			}
		} else if strings.HasPrefix(r, "+") {
			if n := parseRangeStart(r[1:]); n > 0 {
				p.newLine = n
			}
		}
	}
}

// parseRangeStart parses "10,5" or "10" and returns the start number.
func parseRangeStart(s string) int {
	numStr, _, _ := strings.Cut(s, ",")
	n := 0
	for _, ch := range numStr {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

// parseDiffLine converts a raw diff line into a domain.DiffLine.
func (p *diffParser) parseDiffLine(line string) *domain.DiffLine {
	if line == "" {
		dl := &domain.DiffLine{
			Type:    domain.DiffContext,
			Content: "",
			OldNum:  p.oldLine,
			NewNum:  p.newLine,
		}
		p.oldLine++
		p.newLine++
		return dl
	}

	switch line[0] {
	case '+':
		dl := &domain.DiffLine{
			Type:    domain.DiffAdd,
			Content: line[1:],
			NewNum:  p.newLine,
		}
		p.newLine++
		return dl
	case '-':
		dl := &domain.DiffLine{
			Type:    domain.DiffDelete,
			Content: line[1:],
			OldNum:  p.oldLine,
		}
		p.oldLine++
		return dl
	case '\\':
		return nil
	default:
		content := line
		if line[0] == ' ' {
			content = line[1:]
		}
		dl := &domain.DiffLine{
			Type:    domain.DiffContext,
			Content: content,
			OldNum:  p.oldLine,
			NewNum:  p.newLine,
		}
		p.oldLine++
		p.newLine++
		return dl
	}
}
