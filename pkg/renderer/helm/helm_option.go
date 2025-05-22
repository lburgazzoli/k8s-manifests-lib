package helm

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"helm.sh/helm/v3/pkg/cli"
)

// Option defines a functional option for configuring the Helm renderer
type Option func(*Renderer)

// WithFilter adds a filter to the renderer's processing chain
func WithFilter(f types.Filter) Option {
	return func(r *Renderer) {
		r.filters = append(r.filters, f)
	}
}

// WithTransformer adds a transformer to the renderer's processing chain
func WithTransformer(t types.Transformer) Option {
	return func(r *Renderer) {
		r.transformers = append(r.transformers, t)
	}
}

// WithSettings allows customizing the Helm environment settings
func WithSettings(settings *cli.EnvSettings) Option {
	return func(r *Renderer) {
		r.settings = settings
	}
}
