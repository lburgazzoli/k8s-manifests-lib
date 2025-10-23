package mem

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

// RendererOption is a generic option for RendererOptions.
type RendererOption = util.Option[RendererOptions]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
type RendererOptions struct {
	// Filters are renderer-specific filters applied during Process().
	Filters []types.Filter

	// Transformers are renderer-specific transformers applied during Process().
	Transformers []types.Transformer

	// SourceAnnotations enables automatic addition of source tracking annotations.
	SourceAnnotations bool
}

func (opts RendererOptions) ApplyTo(target *RendererOptions) {
	target.Filters = opts.Filters
	target.Transformers = opts.Transformers
	target.SourceAnnotations = opts.SourceAnnotations
}

// WithFilter adds a renderer-specific filter to this Mem renderer's processing chain.
// Renderer-specific filters are applied during Process(), before results are returned to the engine.
// For engine-level filtering applied to all renderers, use engine.WithFilter.
func WithFilter(f types.Filter) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Filters = append(opts.Filters, f)
	})
}

// WithTransformer adds a renderer-specific transformer to this Mem renderer's processing chain.
// Renderer-specific transformers are applied during Process(), before results are returned to the engine.
// For engine-level transformation applied to all renderers, use engine.WithTransformer.
func WithTransformer(t types.Transformer) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Transformers = append(opts.Transformers, t)
	})
}

// WithSourceAnnotations enables or disables automatic addition of source tracking annotations.
// When enabled, the renderer adds metadata annotations to track the source type.
// Annotations added: manifests.k8s-manifests-lib/source.type.
// Default: false (disabled).
func WithSourceAnnotations(enabled bool) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.SourceAnnotations = enabled
	})
}
