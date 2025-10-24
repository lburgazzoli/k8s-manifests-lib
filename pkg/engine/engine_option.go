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
	renderOptions

	renderers []types.Renderer
	parallel  bool
}

// renderOptions represents the processing options for rendering.
type renderOptions struct {
	filters      []types.Filter
	transformers []types.Transformer
	values       map[string]any
}

// EngineOptions is a struct-based option that can set multiple engine options at once.
type EngineOptions struct {
	// Renderers are the manifest sources to process (e.g., Helm, Kustomize, YAML).
	Renderers []types.Renderer

	// Filters are engine-level filters applied to all renders.
	Filters []types.Filter

	// Transformers are engine-level transformers applied to all renders.
	Transformers []types.Transformer

	// Parallel enables parallel execution of renderers.
	Parallel bool
}

func (opts EngineOptions) ApplyTo(target *engineOptions) {
	target.renderers = opts.Renderers
	target.filters = opts.Filters
	target.transformers = opts.Transformers
	target.parallel = opts.Parallel
}

// RenderOptions is a struct-based option that can set multiple render options at once.
type RenderOptions struct {
	// Filters are render-time filters applied only to this specific Render() call.
	// These are merged with (appended to) engine-level filters.
	Filters []types.Filter

	// Transformers are render-time transformers applied only to this specific Render() call.
	// These are merged with (appended to) engine-level transformers.
	Transformers []types.Transformer

	// Values are render-time values passed to all renderers during this specific Render() call.
	// These values are deep merged with Source-level values, with render-time values taking precedence.
	Values map[string]any
}

func (opts RenderOptions) ApplyTo(target *renderOptions) {
	target.filters = append(target.filters, opts.Filters...)
	target.transformers = append(target.transformers, opts.Transformers...)
	if opts.Values != nil {
		target.values = opts.Values
	}
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

// WithFilter adds an engine-level filter function to the processing chain.
// Engine-level filters are applied to aggregated results from all renderers on every Render() call.
// For renderer-specific filtering, use the renderer's WithFilter option (e.g., helm.WithFilter).
// For one-time filtering on a single Render() call, use WithRenderFilter.
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

// WithTransformer adds an engine-level transformer function to the processing chain.
// Engine-level transformers are applied to aggregated results from all renderers on every Render() call.
// For renderer-specific transformation, use the renderer's WithTransformer option (e.g., helm.WithTransformer).
// For one-time transformation on a single Render() call, use WithRenderTransformer.
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

// WithRenderFilter adds a render-time filter function for a single Render() call.
// Render-time filters are merged with (appended to) engine-level filters.
// Use this for one-off filtering that doesn't apply to all renders.
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

// WithRenderTransformer adds a render-time transformer function for a single Render() call.
// Render-time transformers are merged with (appended to) engine-level transformers.
// Use this for one-off transformation that doesn't apply to all renders.
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

// WithParallel enables or disables parallel execution of renderers.
// When enabled, all renderers execute concurrently using goroutines.
// When disabled (default), renderers execute sequentially.
// Parallel execution is beneficial for I/O-bound renderers (Helm OCI fetch, file reads).
func WithParallel(enabled bool) EngineOption {
	return util.FunctionalOption[engineOptions](func(o *engineOptions) {
		o.parallel = enabled
	})
}

// WithValues adds render-time values for a single Render() call.
// These values are passed to all renderers and deep merged with Source-level values,
// with render-time values taking precedence for conflicting keys.
// Renderers that support dynamic values (Helm, Kustomize, GoTemplate) will use these values.
// Renderers that don't support values (YAML, Mem) will ignore them.
func WithValues(values map[string]any) RenderOption {
	return util.FunctionalOption[renderOptions](func(o *renderOptions) {
		o.values = values
	})
}

// RenderValuesOption is a struct-based RenderOption for values.
type RenderValuesOption struct {
	Values map[string]any
}

func (rvo RenderValuesOption) ApplyToRenderOptions(o *renderOptions) {
	o.values = rvo.Values
}
