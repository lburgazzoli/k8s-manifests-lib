package helm

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

// Data represents the input for a Helm rendering operation.
// Path to the chart directory or OCI reference (e.g. "oci://registry/chart:tag" or "/path/to/chart").
// Name of the release.
// Namespace to install the release into.
// Values to apply to the chart.
// The loaded Helm chart.
// The prepared render values.
type Data struct {
	ChartSource  string           // Path to the chart directory or OCI reference (e.g. "oci://registry/chart:tag" or "/path/to/chart")
	ReleaseName  string           // Name of the release
	Namespace    string           // Namespace to install the release into
	Values       map[string]any   // Values to apply to the chart
	chart        *chart.Chart     // The loaded Helm chart
	renderValues chartutil.Values // The prepared render values
}

// Renderer handles Helm rendering operations.
// It implements types.Renderer.
type Renderer struct {
	settings     *cli.EnvSettings
	inputs       []Data
	filters      []types.Filter
	transformers []types.Transformer
	helmEngine   engine.Engine
	decoder      runtime.Serializer
}

// New creates a new Helm Renderer with the given inputs and options.
func New(inputs []Data, opts ...RendererOption) (*Renderer, error) {
	if len(inputs) == 0 {
		return nil, errors.New("at least one input is required")
	}

	r := &Renderer{
		settings:     cli.New(),
		inputs:       slices.Clone(inputs),
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
		helmEngine:   engine.Engine{},
		decoder:      yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme),
	}

	// Apply options
	for _, opt := range opts {
		opt.ApplyTo(r)
	}

	// Validate and prepare charts
	for i, input := range r.inputs {
		if input.ChartSource == "" {
			return nil, fmt.Errorf("input[%d]: ChartSource is required", i)
		}
		if input.ReleaseName == "" {
			return nil, fmt.Errorf("input[%d]: ReleaseName is required", i)
		}
		if input.Namespace == "" {
			return nil, fmt.Errorf("input[%d]: Namespace is required", i)
		}

		// Load the chart
		chart, err := loader.Load(input.ChartSource)
		if err != nil {
			return nil, fmt.Errorf("input[%d]: failed to load chart: %w", i, err)
		}

		// Process dependencies
		if err := chartutil.ProcessDependencies(chart, input.Values); err != nil {
			return nil, fmt.Errorf("input[%d]: failed to process dependencies: %w", i, err)
		}

		// Prepare render values
		renderValues, err := chartutil.ToRenderValues(
			chart,
			input.Values,
			chartutil.ReleaseOptions{
				Name:      input.ReleaseName,
				Namespace: input.Namespace,
				Revision:  1,
				IsInstall: true,
			},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("input[%d]: failed to prepare render values: %w", i, err)
		}

		r.inputs[i].chart = chart
		r.inputs[i].renderValues = renderValues
	}

	return r, nil
}

// Process executes the rendering logic for all configured inputs.
// It implements the types.Renderer interface.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	var allObjects []unstructured.Unstructured

	for i, input := range r.inputs {
		objects, err := r.renderSingle(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error rendering helm chart %s (input[%d]): %w", input.ChartSource, i, err)
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

// renderSingle performs the rendering for a single Helm chart.
// It loads the chart, processes its dependencies, renders the templates,
// and converts the output to unstructured objects.
func (r *Renderer) renderSingle(_ context.Context, data Data) ([]unstructured.Unstructured, error) {
	// Render the chart
	files, err := r.helmEngine.Render(data.chart, data.renderValues)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	// Convert to unstructured objects
	var result []unstructured.Unstructured

	// Process CRDs first
	for _, crd := range data.chart.CRDObjects() {
		objects, err := util.DecodeYAML(r.decoder, crd.File.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode CRD %s: %w", crd.Name, err)
		}
		result = append(result, objects...)
	}

	// Process rendered templates
	for k, v := range files {
		if !strings.HasSuffix(k, ".yaml") && !strings.HasSuffix(k, ".yml") {
			continue
		}
		objects, err := util.DecodeYAML(r.decoder, []byte(v))
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s: %w", k, err)
		}
		result = append(result, objects...)
	}

	return result, nil
}
