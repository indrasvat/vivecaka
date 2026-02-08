package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		got := tt.a.String()
		assert.Equal(t, tt.want, got)
	}
}
