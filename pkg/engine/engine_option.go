package engine

import (
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

// EngineOption is a generic option for engineOptions.
type EngineOption = util.Option[engineOptions]

// RenderOption is a generic option for renderOptions.
type RenderOption = util.Option[renderOptions]

// engineOptions represents the processing options for the engine.
type engineOptions struct {
	renderers []types.Renderer
	renderOptions
}

// renderOptions represents the processing options for rendering.
type renderOptions struct {
	filters      []types.Filter
	transformers []types.Transformer
}

// EngineOptions is a struct-based option that can set multiple engine options at once.
type EngineOptions struct {
	Renderers    []types.Renderer
	Filters      []types.Filter
	Transformers []types.Transformer
}

func (opts EngineOptions) ApplyTo(target *engineOptions) {
	target.renderers = opts.Renderers
	target.filters = opts.Filters
	target.transformers = opts.Transformers
}

// RenderOptions is a struct-based option that can set multiple render options at once.
type RenderOptions struct {
	Filters      []types.Filter
	Transformers []types.Transformer
}

func (opts RenderOptions) ApplyTo(target *renderOptions) {
	target.filters = opts.Filters
	target.transformers = opts.Transformers
}

// WithRenderer adds a configured renderer to the engine.
// Can only be used during engine creation.
func WithRenderer(r types.Renderer) EngineOption {
	return util.FunctionalOption[engineOptions](func(o *engineOptions) {
		o.renderers = append(o.renderers, r)
	})
}

// RendererOption is a struct-based EngineOption.
type RendererOption struct {
	Renderer types.Renderer
}

func (ro RendererOption) ApplyToEngineOptions(o *engineOptions) {
	o.renderers = append(o.renderers, ro.Renderer)
}

// WithFilter adds a filter function to the processing chain.
// Can be used both during engine creation and rendering.
func WithFilter(f types.Filter) EngineOption {
	return util.FunctionalOption[engineOptions](func(o *engineOptions) {
		o.filters = append(o.filters, f)
	})
}

// FilterOption is a struct-based EngineOption.
type FilterOption struct {
	Filter types.Filter
}

func (fo FilterOption) ApplyToEngineOptions(o *engineOptions) {
	o.filters = append(o.filters, fo.Filter)
}

// WithTransformer adds a transformer function to the processing chain.
// Can be used both during engine creation and rendering.
func WithTransformer(t types.Transformer) EngineOption {
	return util.FunctionalOption[engineOptions](func(o *engineOptions) {
		o.transformers = append(o.transformers, t)
	})
}

// TransformerOption is a struct-based EngineOption.
type TransformerOption struct {
	Transformer types.Transformer
}

func (to TransformerOption) ApplyToEngineOptions(o *engineOptions) {
	o.transformers = append(o.transformers, to.Transformer)
}

// WithRenderFilter adds a filter function to be applied during rendering.
func WithRenderFilter(f types.Filter) RenderOption {
	return util.FunctionalOption[renderOptions](func(o *renderOptions) {
		o.filters = append(o.filters, f)
	})
}

// RenderFilterOption is a struct-based RenderOption.
type RenderFilterOption struct {
	Filter types.Filter
}

func (rfo RenderFilterOption) ApplyToRenderOptions(o *renderOptions) {
	o.filters = append(o.filters, rfo.Filter)
}

// WithRenderTransformer adds a transformer function to be applied during rendering.
func WithRenderTransformer(t types.Transformer) RenderOption {
	return util.FunctionalOption[renderOptions](func(o *renderOptions) {
		o.transformers = append(o.transformers, t)
	})
}

// RenderTransformerOption is a struct-based RenderOption.
type RenderTransformerOption struct {
	Transformer types.Transformer
}

func (rto RenderTransformerOption) ApplyToRenderOptions(o *renderOptions) {
	o.transformers = append(o.transformers, rto.Transformer)
}
