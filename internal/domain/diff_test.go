package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		got := tt.t.String()
		assert.Equal(t, tt.want, got)
	}
}
