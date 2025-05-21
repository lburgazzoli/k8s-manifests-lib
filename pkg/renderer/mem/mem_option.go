package mem

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

// Option defines a functional option for the memory renderer.
type Option func(*Renderer)

// WithFilter adds a filter function to the renderer.
func WithFilter(f types.Filter) Option {
	return func(r *Renderer) {
		r.filters = append(r.filters, f)
	}
}

// WithTransformer adds a transformer function to the renderer.
func WithTransformer(t types.Transformer) Option {
	return func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	}
}
