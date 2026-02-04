package domain

import "testing"

func TestRepoRefString(t *testing.T) {
	tests := []struct {
		ref  RepoRef
		want string
	}{
		{RepoRef{Owner: "octocat", Name: "hello-world"}, "octocat/hello-world"},
		{RepoRef{Owner: "org", Name: "repo"}, "org/repo"},
		{RepoRef{}, "/"},
	}
	for _, tt := range tests {
		if got := tt.ref.String(); got != tt.want {
			t.Errorf("RepoRef%+v.String() = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestPRStateString(t *testing.T) {
	tests := []struct {
		s    PRState
		want string
	}{
		{PRStateOpen, "open"},
		{PRStateClosed, "closed"},
		{PRStateMerged, "merged"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("PRState(%q).String() = %q, want %q", string(tt.s), got, tt.want)
		}
	}
}

func TestCIStatusString(t *testing.T) {
	tests := []struct {
		s    CIStatus
		want string
	}{
		{CIPass, "pass"},
		{CIFail, "fail"},
		{CIPending, "pending"},
		{CISkipped, "skipped"},
		{CINone, "none"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("CIStatus(%q).String() = %q, want %q", string(tt.s), got, tt.want)
		}
	}
}

func TestReviewStateString(t *testing.T) {
	tests := []struct {
		s    ReviewState
		want string
	}{
		{ReviewApproved, "approved"},
		{ReviewChangesRequested, "changes_requested"},
		{ReviewPending, "pending"},
		{ReviewNone, "none"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("ReviewState(%q).String() = %q, want %q", string(tt.s), got, tt.want)
		}
	}
}
