package filter

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FilterError represents an error that occurred during filter application.
// It provides context about which object failed and the underlying error.
type FilterError struct {
	Object unstructured.Unstructured
	Err    error
}

func (e *FilterError) Error() string {
	return fmt.Sprintf(
		"filter error for %s:%s %s (namespace: %s): %v",
		e.Object.GroupVersionKind().GroupVersion(),
		e.Object.GroupVersionKind().Kind,
		e.Object.GetName(),
		e.Object.GetNamespace(),
		e.Err,
	)
}

func (e *FilterError) Unwrap() error {
	return e.Err
}

// Error wraps an error with filter context.
// If err is already a FilterError, it returns it as-is to avoid double-wrapping.
// Otherwise, it wraps err in a new FilterError with the provided object context.
func Error(obj unstructured.Unstructured, err error) error {
	if err == nil {
		return nil
	}

	var filterErr *FilterError
	if errors.As(err, &filterErr) {
		return err
	}

	return &FilterError{
		Object: obj,
		Err:    err,
	}
}
