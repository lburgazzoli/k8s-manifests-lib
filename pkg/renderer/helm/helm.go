package helm

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Data represents the input for a Helm rendering operation.
type Data struct {
	PathOrOCIReference string                 // Path to the chart directory or OCI reference
	ReleaseName        string                 // Name of the release
	Namespace          string                 // Namespace to install the release into
	Values             map[string]interface{} // Values to apply to the chart
}

// Renderer handles Helm rendering operations.
// It implements types.Renderer.
type Renderer struct {
	settings     *cli.EnvSettings
	inputs       []Data
	filters      []types.Filter
	transformers []types.Transformer
}

// New creates a new Helm Renderer with the given inputs and options.
func New(inputs []Data, opts ...Option) *Renderer {
	r := &Renderer{
		settings:     cli.New(),
		inputs:       inputs,
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Process executes the rendering logic for all configured inputs.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	var allObjects []unstructured.Unstructured

	for _, input := range r.inputs {
		objects, err := r.renderSingle(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error rendering helm chart %s: %w", input.PathOrOCIReference, err)
		}
		allObjects = append(allObjects, objects...)
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
