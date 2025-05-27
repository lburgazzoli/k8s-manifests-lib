package mem

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

// RendererOption is a generic option for Renderer.
type RendererOption = util.Option[Renderer]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
type RendererOptions struct {
	Filters      []types.Filter
	Transformers []types.Transformer
}

func (opts RendererOptions) ApplyTo(target *Renderer) {
	target.filters = opts.Filters
	target.transformers = opts.Transformers
}

// WithFilter adds a filter function to the renderer.
func WithFilter(f types.Filter) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.filters = append(r.filters, f)
	})
}

// WithTransformer adds a transformer function to the renderer.
func WithTransformer(t types.Transformer) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	})
}
