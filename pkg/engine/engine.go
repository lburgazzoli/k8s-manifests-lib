package engine

import (
	"context"
	"fmt"
	"slices"
	"sync"

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

	var allObjects []unstructured.Unstructured
	var err error

	// Process renderers in parallel or sequentially
	if e.options.parallel {
		allObjects, err = e.renderParallel(ctx)
	} else {
		allObjects, err = e.renderSequential(ctx)
	}

	if err != nil {
		return nil, err
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

// renderSequential processes renderers sequentially in order.
func (e *Engine) renderSequential(ctx context.Context) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for i, renderer := range e.options.renderers {
		objects, err := renderer.Process(ctx)
		if err != nil {
			return nil, fmt.Errorf("error processing renderer[%d] (%T): %w", i, renderer, err)
		}

		allObjects = append(allObjects, objects...)
	}

	return allObjects, nil
}

// renderParallel processes all renderers concurrently using goroutines.
func (e *Engine) renderParallel(ctx context.Context) ([]unstructured.Unstructured, error) {
	type result struct {
		objects []unstructured.Unstructured
		err     error
		index   int
	}

	results := make(chan result, len(e.options.renderers))
	var wg sync.WaitGroup

	for i, renderer := range e.options.renderers {
		wg.Add(1)
		go func(idx int, r types.Renderer) {
			defer wg.Done()
			objects, err := r.Process(ctx)
			results <- result{objects: objects, err: err, index: idx}
		}(i, renderer)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	allObjects := make([]unstructured.Unstructured, 0)
	for res := range results {
		if res.err != nil {
			return nil, fmt.Errorf("error processing renderer[%d] (%T): %w", res.index, e.options.renderers[res.index], res.err)
		}
		allObjects = append(allObjects, res.objects...)
	}

	return allObjects, nil
}
