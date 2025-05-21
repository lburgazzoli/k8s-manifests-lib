// Package mem provides a memory-based renderer for Kubernetes manifests.
// It handles rendering of unstructured objects that are already in memory.
package mem

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Renderer handles memory-based rendering operations.
// It implements types.Renderer for objects that are already in memory.
type Renderer struct {
	objects      []unstructured.Unstructured
	filters      []types.Filter
	transformers []types.Transformer
}

// NewRenderer creates a new memory-based renderer with the given objects and options.
func NewRenderer(objects []unstructured.Unstructured, opts ...Option) *Renderer {
	r := &Renderer{
		objects:      objects,
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Process implements types.Renderer by returning the objects that were provided during construction.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	// Make deep copies of all objects
	allObjects := make([]unstructured.Unstructured, 0, len(r.objects))
	for _, obj := range r.objects {
		allObjects = append(allObjects, *obj.DeepCopy())
	}

	// Apply filters
	filtered, err := util.ApplyFilters(ctx, allObjects, r.filters)
	if err != nil {
		return nil, fmt.Errorf("error applying filters: %w", err)
	}

	// Apply transformers
	transformed, err := util.ApplyTransformers(ctx, filtered, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("error applying transformers: %w", err)
	}

	return transformed, nil
}
