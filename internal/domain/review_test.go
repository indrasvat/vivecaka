package domain

import "testing"

func TestReviewActionString(t *testing.T) {
	tests := []struct {
		a    ReviewAction
		want string
	}{
		{ReviewActionApprove, "approve"},
		{ReviewActionRequestChanges, "request_changes"},
		{ReviewActionComment, "comment"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("ReviewAction(%q).String() = %q, want %q", string(tt.a), got, tt.want)
		}
	}
}
