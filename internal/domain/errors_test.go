package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrNotFound", ErrNotFound, "not found"},
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized"},
		{"ErrNotAuthenticated", ErrNotAuthenticated, "not authenticated: run 'gh auth login'"},
		{"ErrRateLimited", ErrRateLimited, "rate limited"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSentinelErrorsIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
	}{
		{"ErrNotFound", ErrNotFound, ErrNotFound},
		{"ErrUnauthorized", ErrUnauthorized, ErrUnauthorized},
		{"ErrNotAuthenticated", ErrNotAuthenticated, ErrNotAuthenticated},
		{"ErrRateLimited", ErrRateLimited, ErrRateLimited},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, errors.Is(tt.err, tt.target))
		})
	}
}

func TestSentinelErrorsWrapped(t *testing.T) {
	wrapped := fmt.Errorf("fetching PR #42: %w", ErrNotFound)
	assert.True(t, errors.Is(wrapped, ErrNotFound))
	assert.False(t, errors.Is(wrapped, ErrUnauthorized))
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "title", Message: "cannot be empty"}
	want := "validation error on title: cannot be empty"
	got := err.Error()
	assert.Equal(t, want, got)
}

func TestValidationErrorAs(t *testing.T) {
	original := &ValidationError{Field: "body", Message: "too long"}
	wrapped := fmt.Errorf("invalid PR: %w", original)

	var ve *ValidationError
	require.True(t, errors.As(wrapped, &ve))
	assert.Equal(t, "body", ve.Field)
	assert.Equal(t, "too long", ve.Message)
}

func TestValidationErrorNotMatchSentinel(t *testing.T) {
	ve := &ValidationError{Field: "x", Message: "y"}
	assert.False(t, errors.Is(ve, ErrNotFound))
}
