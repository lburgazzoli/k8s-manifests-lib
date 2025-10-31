package mem

import (
	"fmt"
)

// sourceHolder wraps a Source with internal state for consistency with other renderers.
type sourceHolder struct {
	Source
}

// Validate checks if the Source configuration is valid.
func (h *sourceHolder) Validate() error {
	for i := range h.Objects {
		if len(h.Objects[i].Object) == 0 {
			return fmt.Errorf("object at index %d is empty or has nil internal data", i)
		}
	}
	return nil
}
