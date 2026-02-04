package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
var (
	ErrNotFound         = errors.New("not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNotAuthenticated = errors.New("not authenticated: run 'gh auth login'")
	ErrRateLimited      = errors.New("rate limited")
)

// ValidationError represents a validation failure on a specific field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}
