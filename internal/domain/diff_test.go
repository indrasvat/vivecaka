package domain

import "testing"

func TestDiffLineTypeString(t *testing.T) {
	tests := []struct {
		t    DiffLineType
		want string
	}{
		{DiffAdd, "add"},
		{DiffDelete, "delete"},
		{DiffContext, "context"},
	}
	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("DiffLineType(%q).String() = %q, want %q", string(tt.t), got, tt.want)
		}
	}
}
