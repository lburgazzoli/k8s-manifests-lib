package engine

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics"
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
	startTime := time.Now()

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
	filtered, err := pipeline.ApplyFilters(ctx, allObjects, renderOpts.filters)
	if err != nil {
		return nil, fmt.Errorf("engine filter error: %w", err)
	}

	// Apply transformers
	transformed, err := pipeline.ApplyTransformers(ctx, filtered, renderOpts.transformers)
	if err != nil {
		return nil, fmt.Errorf("engine transformer error: %w", err)
	}

	metrics.ObserveRender(ctx, time.Since(startTime), len(transformed))

	return transformed, nil
}

// processRenderer executes a single renderer with timing, metrics, and error handling.
func (e *Engine) processRenderer(ctx context.Context, renderer types.Renderer) ([]unstructured.Unstructured, error) {
	startTime := time.Now()
	objects, err := renderer.Process(ctx)

	metrics.ObserveRenderer(ctx, renderer.Name(), time.Since(startTime), len(objects), err)

	if err != nil {
		return nil, fmt.Errorf("error processing renderer %q (%T): %w", renderer.Name(), renderer, err)
	}

	return objects, nil
}

// renderSequential processes renderers sequentially in order.
func (e *Engine) renderSequential(ctx context.Context) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for _, renderer := range e.options.renderers {
		objects, err := e.processRenderer(ctx, renderer)
		if err != nil {
			return nil, err
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
	}

	results := make(chan result, len(e.options.renderers))
	var wg sync.WaitGroup

	for _, renderer := range e.options.renderers {
		wg.Add(1)
		go func(r types.Renderer) {
			defer wg.Done()
			objects, err := e.processRenderer(ctx, r)
			results <- result{
				objects: objects,
				err:     err,
			}
		}(renderer)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	allObjects := make([]unstructured.Unstructured, 0)
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}

		allObjects = append(allObjects, res.objects...)
	}

	return allObjects, nil
}
