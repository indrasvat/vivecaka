package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		got := tt.ref.String()
		assert.Equal(t, tt.want, got)
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
		got := tt.s.String()
		assert.Equal(t, tt.want, got)
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
		got := tt.s.String()
		assert.Equal(t, tt.want, got)
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
		got := tt.s.String()
		assert.Equal(t, tt.want, got)
	}
}
