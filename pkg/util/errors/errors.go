// Package errors provides common error definitions used across the project.
package errors

import "errors"

var (
	// ErrFsRequired is returned when a filesystem is required but not provided.
	ErrFsRequired = errors.New("fs is required")

	// ErrPathEmpty is returned when a path is empty or contains only whitespace.
	ErrPathEmpty = errors.New("path cannot be empty or whitespace-only")
)
