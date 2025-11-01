package yaml

import (
	"strings"

	utilerrors "github.com/lburgazzoli/k8s-manifests-lib/pkg/util/errors"
)

// sourceHolder wraps a Source with internal state for consistency with other renderers.
type sourceHolder struct {
	Source
}

// Validate checks if the Source configuration is valid.
func (h *sourceHolder) Validate() error {
	if h.FS == nil {
		return utilerrors.ErrFsRequired
	}
	if len(strings.TrimSpace(h.Path)) == 0 {
		return utilerrors.ErrPathEmpty
	}

	return nil
}
