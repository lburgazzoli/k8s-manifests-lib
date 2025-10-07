package engine

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

// Engine represents the core manifest rendering and processing engine.
type Engine struct {
	options engineOptions
}

// New creates a new Engine with the given options.
func New(opts ...EngineOption) *Engine {
	options := engineOptions{
		renderers: make([]types.Renderer, 0),
		renderOptions: renderOptions{
			filters:      make([]types.Filter, 0),
			transformers: make([]types.Transformer, 0),
		},
	}

	for _, opt := range opts {
		opt.ApplyTo(&options)
	}

	return &Engine{
		options: options,
	}
}

// Render processes all inputs associated with the registered renderer configurations
// and returns a consolidated slice of unstructured.Unstructured objects.
//
// The rendering pipeline has three stages for filters and transformers:
//  1. renderer-specific: Each renderer applies its own filters/transformers during Process()
//  2. engine-level: Filters/transformers configured via New() are applied to aggregated results
//  3. render-time: Filters/transformers passed via opts are merged with engine-level ones
//
// Render-time options are additive - they append to engine-level options.
func (e *Engine) Render(ctx context.Context, opts ...RenderOption) ([]unstructured.Unstructured, error) {
	// Initialize render options by cloning the engine's options
	renderOpts := renderOptions{
		filters:      slices.Clone(e.options.filters),
		transformers: slices.Clone(e.options.transformers),
	}

	// Apply render options
	for _, opt := range opts {
		opt.ApplyTo(&renderOpts)
	}

	allObjects := make([]unstructured.Unstructured, 0)

	// Process each renderer
	for i, renderer := range e.options.renderers {
		objects, err := renderer.Process(ctx)
		if err != nil {
			return nil, fmt.Errorf("error processing renderer[%d] (%T): %w", i, renderer, err)
		}

		allObjects = append(allObjects, objects...)
	}

	// Apply filters
	filtered, err := util.ApplyFilters(ctx, allObjects, renderOpts.filters)
	if err != nil {
		return nil, fmt.Errorf("engine filter error: %w", err)
	}

	// Apply transformers
	transformed, err := util.ApplyTransformers(ctx, filtered, renderOpts.transformers)
	if err != nil {
		return nil, fmt.Errorf("engine transformer error: %w", err)
	}

	return transformed, nil
}
