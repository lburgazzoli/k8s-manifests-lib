package gotemplate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

// Values returns a Values function that always returns the provided static values.
// This is a convenience helper for the common case of non-dynamic values.
func Values(values any) func(context.Context) (any, error) {
	return func(_ context.Context) (any, error) {
		return values, nil
	}
}

// sourceHolder wraps a Source with internal state for lazy loading and thread-safety.
type sourceHolder struct {
	Source

	// Mutex protects concurrent access to templates field
	mu *sync.RWMutex

	// Parsed templates (lazy-loaded on first Process call, protected by mu)
	templates *template.Template
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

// LoadTemplates returns parsed templates, loading them lazily if needed.
// Thread-safe for concurrent use.
func (h *sourceHolder) LoadTemplates() (*template.Template, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.templates != nil {
		return h.templates, nil
	}

	tmpl, err := template.ParseFS(h.FS, h.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates (path: %s): %w", h.Path, err)
	}

	h.templates = tmpl.Option("missingkey=error")
	return h.templates, nil
}
