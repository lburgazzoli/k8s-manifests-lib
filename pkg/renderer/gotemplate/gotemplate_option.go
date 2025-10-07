package gotemplate

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
)

// Option defines a functional option for the GoTemplate Renderer.
type Option func(*Renderer)

// RendererOption is a generic option for Renderer.
type RendererOption = util.Option[Renderer]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
type RendererOptions struct {
	// Filters are renderer-specific filters applied during Process().
	Filters []types.Filter

	// Transformers are renderer-specific transformers applied during Process().
	Transformers []types.Transformer

	// Cache is a custom cache implementation for render results.
	Cache cache.Interface[[]unstructured.Unstructured]
}

func (opts RendererOptions) ApplyTo(target *Renderer) {
	target.filters = opts.Filters
	target.transformers = opts.Transformers

	if opts.Cache != nil {
		target.cache = opts.Cache
	}
}

// WithFilter adds a renderer-specific filter to this GoTemplate renderer's processing chain.
// Renderer-specific filters are applied during Process(), before results are returned to the engine.
// For engine-level filtering applied to all renderers, use engine.WithFilter.
func WithFilter(f types.Filter) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.filters = append(r.filters, f)
	})
}

// WithTransformer adds a renderer-specific transformer to this GoTemplate renderer's processing chain.
// Renderer-specific transformers are applied during Process(), before results are returned to the engine.
// For engine-level transformation applied to all renderers, use engine.WithTransformer.
func WithTransformer(t types.Transformer) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	})
}

// WithCache enables render result caching with the specified options.
// If no options are provided, uses default TTL of 5 minutes.
// By default, caching is NOT enabled.
func WithCache(opts ...cache.Option) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.cache = cache.NewRenderCache(opts...)
	})
}
