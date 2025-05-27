package helm

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"helm.sh/helm/v3/pkg/cli"
)

// RendererOption is a generic option for Renderer.
type RendererOption = util.Option[Renderer]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
type RendererOptions struct {
	Filters      []types.Filter
	Transformers []types.Transformer
	Settings     **cli.EnvSettings
}

func (opts RendererOptions) ApplyTo(target *Renderer) {
	target.filters = opts.Filters
	target.transformers = opts.Transformers
	if opts.Settings != nil {
		target.settings = *opts.Settings
	}
}

// WithFilter adds a filter to the renderer's processing chain.
func WithFilter(f types.Filter) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.filters = append(r.filters, f)
	})
}

// WithTransformer adds a transformer to the renderer's processing chain.
func WithTransformer(t types.Transformer) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	})
}

// WithSettings allows customizing the Helm environment settings.
func WithSettings(settings *cli.EnvSettings) RendererOption {
	return util.FunctionalOption[Renderer](func(r *Renderer) {
		r.settings = settings
	})
}
