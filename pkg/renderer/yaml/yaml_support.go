package yaml

import (
	"errors"
	"strings"
)

// sourceHolder wraps a Source with internal state for consistency with other renderers.
type sourceHolder struct {
	Source
}

// Validate checks if the Source configuration is valid.
func (h *sourceHolder) Validate() error {
	if h.FS == nil {
		return errors.New("fs is required")
	}
	if len(strings.TrimSpace(h.Path)) == 0 {
		return errors.New("path cannot be empty or whitespace-only")
	}
	return nil
}
