package engine

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

// Option defines a functional option for the Engine.
type Option func(*Engine)

// WithRenderer adds a configured renderer to the engine.
func WithRenderer(r types.Renderer) Option {
	return func(e *Engine) {
		e.renderers = append(e.renderers, r)
	}
}

// WithFilter adds a filter function to the engine's processing chain.
func WithFilter(f types.Filter) Option {
	return func(e *Engine) {
		e.filters = append(e.filters, f)
	}
}

// WithTransformer adds a transformer function to the engine's processing chain.
func WithTransformer(t types.Transformer) Option {
	return func(e *Engine) {
		e.transformers = append(e.transformers, t)
	}
}
