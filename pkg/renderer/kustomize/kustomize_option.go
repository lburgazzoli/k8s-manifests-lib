package kustomize

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

// RendererOption is a generic option for Renderer.
type RendererOption = util.Option[Renderer]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
type RendererOptions struct {
	Filters          []types.Filter
	Transformers     []types.Transformer
	Plugins          []resmap.Transformer
	LoadRestrictions *kustomizetypes.LoadRestrictions
}

func (opts RendererOptions) ApplyTo(target *Renderer) {
	target.filters = opts.Filters
	target.transformers = opts.Transformers
	target.plugins = opts.Plugins
	if opts.LoadRestrictions != nil {
		target.kustomizeOpts.LoadRestrictions = *opts.LoadRestrictions
	}
}

// WithFilter adds a renderer-specific filter function.
func WithFilter(f types.Filter) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.filters = append(r.filters, f)
	})
}

// WithTransformer adds a transformer (types.Transformer) to the renderer.
func WithTransformer(t types.Transformer) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	})
}

// WithPlugin registers a plugin transformer (resmap.Transformer) for kustomize.
func WithPlugin(plugin resmap.Transformer) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.plugins = append(r.plugins, plugin)
	})
}

// WithLoadRestrictions sets the load restrictions for kustomize.
func WithLoadRestrictions(restrictions kustomizetypes.LoadRestrictions) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.kustomizeOpts.LoadRestrictions = restrictions
	})
}
