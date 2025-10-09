package helm

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/registry"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/dump"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/k8s"
)

const rendererType = "helm"

type Source struct {
	// Repo is the repository URL for chart lookup. Optional for local or OCI charts.
	Repo string

	// Chart specifies the chart to render. Supports OCI references (oci://registry/chart:tag)
	// or local filesystem paths. Required.
	Chart string

	// ReleaseName is the Helm release name used in template rendering metadata.
	// Required for proper .Release.Name substitution in templates.
	ReleaseName string

	// ReleaseVersion constrains the chart version to fetch. Optional; uses latest if empty.
	ReleaseVersion string

	// Values provides template variable overrides during chart rendering.
	// Function is called during rendering to obtain dynamic values.
	// Merged with chart defaults via chartutil.ToRenderValues.
	Values func(context.Context) (map[string]any, error)

	// ProcessDependencies determines whether chart dependencies should be processed.
	// If true, chartutil.ProcessDependencies will be called during rendering.
	// Default is false.
	ProcessDependencies bool

	// The loaded Helm chart
	chart *chart.Chart
}

// Values returns a Values function that always returns the provided static values.
// This is a convenience helper for the common case of non-dynamic values.
func Values(values map[string]any) func(context.Context) (map[string]any, error) {
	return func(_ context.Context) (map[string]any, error) {
		return values, nil
	}
}

// Renderer handles Helm rendering operations.
// It implements types.Renderer.
type Renderer struct {
	settings     *cli.EnvSettings
	install      *action.Install
	inputs       []Source
	filters      []types.Filter
	transformers []types.Transformer
	helmEngine   engine.Engine
	cache        cache.Interface[[]unstructured.Unstructured]
}

// New creates a new Helm Renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	// Validate inputs
	for i, input := range inputs {
		if input.Chart == "" {
			return nil, fmt.Errorf("input[%d]: Chart is required", i)
		}
		if input.ReleaseName == "" {
			return nil, fmt.Errorf("input[%d]: ReleaseName is required", i)
		}
	}

	r := &Renderer{
		settings:     nil,
		inputs:       slices.Clone(inputs),
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
		helmEngine:   engine.Engine{},
	}

	// Apply options
	for _, opt := range opts {
		opt.ApplyTo(r)
	}

	if r.settings == nil {
		r.settings = cli.New()
	}

	c, err := registry.NewClient()
	if err != nil {
		return nil, fmt.Errorf("unable to create a registry client: %w", err)
	}

	r.install = action.NewInstall(&action.Configuration{
		RegistryClient: c,
	})

	return r, nil
}

// Process executes the rendering logic for all configured inputs.
// It implements the types.Renderer interface.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for i := range r.inputs {
		// Load chart if not already loaded (lazy loading for retry support)
		if r.inputs[i].chart == nil {
			opt := r.install.ChartPathOptions
			opt.RepoURL = r.inputs[i].Repo
			opt.Version = r.inputs[i].ReleaseVersion

			path, err := opt.LocateChart(r.inputs[i].Chart, r.settings)
			if err != nil {
				return nil, fmt.Errorf(
					"input[%d]: unable to locate chart (repo: %s, name: %s, version: %s): %w",
					i,
					r.inputs[i].Repo,
					r.inputs[i].Chart,
					r.inputs[i].ReleaseVersion,
					err)
			}

			c, err := loader.Load(path)
			if err != nil {
				return nil, fmt.Errorf(
					"input[%d]: failed to load chart (repo: %s, name: %s, version: %s): %w",
					i,
					r.inputs[i].Repo,
					r.inputs[i].Chart,
					r.inputs[i].ReleaseVersion,
					err,
				)
			}

			r.inputs[i].chart = c
		}

		objects, err := r.renderSingle(ctx, r.inputs[i])
		if err != nil {
			return nil, fmt.Errorf(
				"error rendering helm chart[%d] %s (release: %s): %w",
				i,
				r.inputs[i].Chart,
				r.inputs[i].ReleaseName,
				err,
			)
		}

		allObjects = append(allObjects, objects...)
	}

	transformed, err := pipeline.Apply(ctx, allObjects, r.filters, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("helm renderer: %w", err)
	}

	return transformed, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

func (r *Renderer) values(ctx context.Context, input Source) (map[string]any, error) {
	if input.Values == nil {
		return map[string]any{}, nil
	}

	v, err := input.Values(ctx)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// prepareRenderValues gets values from the Values function, processes dependencies,
// and prepares render values using chartutil.ToRenderValues.
func (r *Renderer) prepareRenderValues(ctx context.Context, input Source) (chartutil.Values, error) {
	// Get values dynamically
	values, err := r.values(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get values: %w", err)
	}

	// Process dependencies if enabled
	if input.ProcessDependencies {
		if err := chartutil.ProcessDependencies(input.chart, values); err != nil {
			return nil, fmt.Errorf("failed to process dependencies: %w", err)
		}
	}

	// Prepare render values
	renderValues, err := chartutil.ToRenderValues(
		input.chart,
		values,
		chartutil.ReleaseOptions{
			Name:      input.ReleaseName,
			Revision:  1,
			IsInstall: true,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare render values: %w", err)
	}

	return renderValues, nil
}

// renderSingle performs the rendering for a single Helm chart.
// It processes dependencies, prepares render values, renders the templates,
// and converts the output to unstructured objects.
func (r *Renderer) renderSingle(ctx context.Context, input Source) ([]unstructured.Unstructured, error) {
	// Prepare render values
	renderValues, err := r.prepareRenderValues(ctx, input)
	if err != nil {
		return nil, err
	}

	// Compute cache key from render values
	var cacheKey string

	// Check cache (if enabled)
	if r.cache != nil {
		cacheKey = dump.ForHash(renderValues)

		// ensure objects are evicted
		r.cache.Sync()

		if cached, found := r.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Render the chart
	files, err := r.helmEngine.Render(input.chart, renderValues)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	result := make([]unstructured.Unstructured, 0)

	// Process CRDs first
	for _, crd := range input.chart.CRDObjects() {
		objects, err := k8s.DecodeYAML(crd.File.Data)
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

		objects, err := k8s.DecodeYAML([]byte(v))
		if err != nil {
			return nil, fmt.Errorf("failed to decode %s: %w", k, err)
		}

		result = append(result, objects...)
	}

	// Cache result (if enabled)
	if r.cache != nil {
		r.cache.Set(cacheKey, result)
	}

	return result, nil
}
