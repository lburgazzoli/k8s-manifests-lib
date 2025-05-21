package kustomize

import "github.com/lburgazzoli/k8s-manifests-lib/pkg/types"

// Option defines a functional option for the Kustomize Renderer.
type Option func(*Renderer)

// WithFilter adds a renderer-specific filter function.
func WithFilter(f types.Filter) Option {
	return func(r *Renderer) {
		r.filters = append(r.filters, f)
	}
}

// WithTransformer adds a renderer-specific transformer function.
func WithTransformer(t types.Transformer) Option {
	return func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	}
}
