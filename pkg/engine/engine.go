package engine

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Engine represents the core manifest rendering and processing engine.
type Engine struct {
	renderers    []types.Renderer
	filters      []types.Filter
	transformers []types.Transformer
}

// New creates a new Engine with the given options.
func New(opts ...Option) *Engine {
	e := &Engine{
		renderers:    make([]types.Renderer, 0),
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Render processes all inputs associated with the registered renderer configurations
// and returns a consolidated slice of unstructured.Unstructured objects.
func (e *Engine) Render(ctx context.Context) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	// Process each renderer
	for i, renderer := range e.renderers {
		objects, err := renderer.Process(ctx)
		if err != nil {
			return nil, fmt.Errorf("error processing renderer #%d: %w", i, err)
		}
		allObjects = append(allObjects, objects...)
	}

	// Apply filters
	fo, err := util.ApplyFilters(ctx, allObjects, e.filters)
	if err != nil {
		return nil, fmt.Errorf("error applying filters: %w", err)
	}

	// Apply transformers
	to, err := util.ApplyTransformers(ctx, fo, e.transformers)
	if err != nil {
		return nil, fmt.Errorf("error applying transformers: %w", err)
	}

	return to, nil
}
