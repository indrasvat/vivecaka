package domain

import (
	"errors"
	"fmt"
	"testing"
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
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("%s.Error() = %q, want %q", tt.name, got, tt.want)
			}
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
			if !errors.Is(tt.err, tt.target) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.target)
			}
		})
	}
}

func TestSentinelErrorsWrapped(t *testing.T) {
	wrapped := fmt.Errorf("fetching PR #42: %w", ErrNotFound)
	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("errors.Is(wrapped, ErrNotFound) = false, want true")
	}
	if errors.Is(wrapped, ErrUnauthorized) {
		t.Error("errors.Is(wrapped, ErrUnauthorized) = true, want false")
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "title", Message: "cannot be empty"}
	want := "validation error on title: cannot be empty"
	if got := err.Error(); got != want {
		t.Errorf("ValidationError.Error() = %q, want %q", got, want)
	}
}

func TestValidationErrorAs(t *testing.T) {
	original := &ValidationError{Field: "body", Message: "too long"}
	wrapped := fmt.Errorf("invalid PR: %w", original)

	var ve *ValidationError
	if !errors.As(wrapped, &ve) {
		t.Fatal("errors.As(wrapped, *ValidationError) = false, want true")
	}
	if ve.Field != "body" {
		t.Errorf("ve.Field = %q, want %q", ve.Field, "body")
	}
	if ve.Message != "too long" {
		t.Errorf("ve.Message = %q, want %q", ve.Message, "too long")
	}
}

func TestValidationErrorNotMatchSentinel(t *testing.T) {
	ve := &ValidationError{Field: "x", Message: "y"}
	if errors.Is(ve, ErrNotFound) {
		t.Error("ValidationError should not match ErrNotFound")
	}
}
