package transformer

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// TransformerError represents an error that occurred during transformer application.
// It provides context about which object failed and the underlying error.
type TransformerError struct {
	Object unstructured.Unstructured
	Err    error
}

func (e *TransformerError) Error() string {
	return fmt.Sprintf(
		"transformer error for %s:%s %s (namespace: %s): %v",
		e.Object.GroupVersionKind().GroupVersion(),
		e.Object.GroupVersionKind().Kind,
		e.Object.GetName(),
		e.Object.GetNamespace(),
		e.Err,
	)
}

func (e *TransformerError) Unwrap() error {
	return e.Err
}

// Error wraps an error with transformer context.
// If err is already a TransformerError, it returns it as-is to avoid double-wrapping.
// Otherwise, it wraps err in a new TransformerError with the provided object context.
func Error(obj unstructured.Unstructured, err error) error {
	if err == nil {
		return nil
	}

	var transformerErr *TransformerError
	if errors.As(err, &transformerErr) {
		return err
	}

	return &TransformerError{
		Object: obj,
		Err:    err,
	}
}
