// Package mem provides a memory-based renderer for Kubernetes manifests.
// It handles rendering of unstructured objects that are already in memory.
package mem

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

const rendererType = "mem"

// Source represents the input for a memory-based rendering operation.
type Source struct {
	// Objects contains pre-constructed Kubernetes manifests to pass through.
	// Useful for testing, composition, or when objects are already in memory.
	Objects []unstructured.Unstructured
}

// Renderer handles memory-based rendering operations.
// It implements types.Renderer for objects that are already in memory.
type Renderer struct {
	inputs       []Source
	filters      []types.Filter
	transformers []types.Transformer
}

// New creates a new memory-based renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	r := &Renderer{
		inputs:       slices.Clone(inputs),
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
	}

	for _, opt := range opts {
		opt.ApplyTo(r)
	}

	return r, nil
}

// Process implements types.Renderer by returning the objects that were provided during construction.
// Render-time values are ignored by the memory renderer as objects are already constructed.
func (r *Renderer) Process(ctx context.Context, _ map[string]any) ([]unstructured.Unstructured, error) {
	// Make deep copies of all objects from all inputs
	allObjects := make([]unstructured.Unstructured, 0)
	for _, input := range r.inputs {
		for _, obj := range input.Objects {
			allObjects = append(allObjects, *obj.DeepCopy())
		}
	}

	transformed, err := pipeline.Apply(ctx, allObjects, r.filters, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("mem renderer: %w", err)
	}

	return transformed, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}
