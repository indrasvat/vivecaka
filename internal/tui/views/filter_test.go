package views

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/indrasvat/vivecaka/internal/domain"
)

func TestFilterDefaults(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	opts := m.Opts()

	if opts.State != domain.PRStateOpen {
		t.Errorf("default state = %q, want %q", opts.State, domain.PRStateOpen)
	}
	if opts.Author != "" {
		t.Errorf("default author = %q, want empty", opts.Author)
	}
	if len(opts.Labels) != 0 {
		t.Errorf("default labels = %v, want empty", opts.Labels)
	}
	if opts.CI != "" {
		t.Errorf("default CI = %q, want empty", opts.CI)
	}
	if opts.Review != "" {
		t.Errorf("default review = %q, want empty", opts.Review)
	}
	if opts.Draft != domain.DraftInclude {
		t.Errorf("default draft = %q, want %q", opts.Draft, domain.DraftInclude)
	}
}

func TestFilterSetOpts(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.SetOpts(domain.ListOpts{
		State:  domain.PRStateClosed,
		Author: "alice",
		Labels: []string{"bug"},
		CI:     domain.CIFail,
		Review: domain.ReviewPending,
		Draft:  domain.DraftOnly,
	})

	opts := m.Opts()
	if opts.State != domain.PRStateClosed {
		t.Errorf("state = %q, want %q", opts.State, domain.PRStateClosed)
	}
	if opts.Author != "alice" {
		t.Errorf("author = %q, want %q", opts.Author, "alice")
	}
	if !reflect.DeepEqual(opts.Labels, []string{"bug"}) {
		t.Errorf("labels = %v, want [bug]", opts.Labels)
	}
	if opts.CI != domain.CIFail {
		t.Errorf("ci = %q, want %q", opts.CI, domain.CIFail)
	}
	if opts.Review != domain.ReviewPending {
		t.Errorf("review = %q, want %q", opts.Review, domain.ReviewPending)
	}
	if opts.Draft != domain.DraftOnly {
		t.Errorf("draft = %q, want %q", opts.Draft, domain.DraftOnly)
	}
}

func TestFilterApplyMessage(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.focus = filterFieldApply

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected apply command")
	}
	if _, ok := cmd().(ApplyFilterMsg); !ok {
		t.Fatalf("expected ApplyFilterMsg, got %T", cmd())
	}
}

func TestFilterCancelMessage(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected close command")
	}
	if _, ok := cmd().(CloseFilterMsg); !ok {
		t.Fatalf("expected CloseFilterMsg, got %T", cmd())
	}
}

func TestFilterResetKey(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.statusIdx = 2
	m.author = "bob"
	m.labelSelected["bug"] = true
	m.ciIdx = 2
	m.reviewIdx = 2
	m.draftIdx = 2

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	opts := m.Opts()
	if opts.State != domain.PRStateOpen || opts.Author != "" || len(opts.Labels) != 0 {
		t.Errorf("reset did not clear filters: %+v", opts)
	}
	if opts.Draft != domain.DraftInclude {
		t.Errorf("draft after reset = %q, want %q", opts.Draft, domain.DraftInclude)
	}
}

func TestFilterLabelToggle(t *testing.T) {
	m := NewFilterModel(testStyles(), testKeys())
	m.focus = filterFieldLabel

	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	opts := m.Opts()
	if !reflect.DeepEqual(opts.Labels, []string{"enhancement"}) {
		t.Errorf("labels after toggle = %v, want [enhancement]", opts.Labels)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	opts = m.Opts()
	if !reflect.DeepEqual(opts.Labels, []string{"enhancement", "bug"}) {
		t.Errorf("labels after second toggle = %v, want [enhancement bug]", opts.Labels)
	}
}
