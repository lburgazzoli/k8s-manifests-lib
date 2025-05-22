package kustomize

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

// Option defines a functional option for the Kustomize Renderer.
type Option func(*Renderer)

// WithFilter adds a renderer-specific filter function.
func WithFilter(f types.Filter) Option {
	return func(r *Renderer) {
		r.filters = append(r.filters, f)
	}
}

// WithTransformer adds a transformer (types.Transformer) to the renderer.
func WithTransformer(t types.Transformer) Option {
	return func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	}
}

// WithPlugin registers a plugin transformer (resmap.Transformer) for kustomize.
func WithPlugin(plugin resmap.Transformer) Option {
	return func(r *Renderer) {
		r.plugins = append(r.plugins, plugin)
	}
}

// WithLoadRestrictions sets the load restrictions for kustomize.
func WithLoadRestrictions(restrictions kustomizetypes.LoadRestrictions) Option {
	return func(r *Renderer) {
		r.kustomizeOpts.LoadRestrictions = restrictions
	}
}
