package errors

import "errors"

// Common validation errors used across renderers.
var (
	// ErrFsRequired is returned when a required filesystem is nil.
	ErrFsRequired = errors.New("fs is required")

	// ErrPathEmpty is returned when a required path is empty or whitespace-only.
	ErrPathEmpty = errors.New("path cannot be empty or whitespace-only")
)

