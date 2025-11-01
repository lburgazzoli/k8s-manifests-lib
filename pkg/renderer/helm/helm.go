package helm

import (
	"context"
	"fmt"
	"sync"

	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/engine"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/dump"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

const rendererType = "helm"

// Source defines a Helm chart source for rendering.
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
}

// Renderer handles Helm rendering operations.
// It implements types.Renderer.
//
// Thread-safety: Renderer is safe for concurrent use. Multiple goroutines
// may call Process() concurrently on the same Renderer instance. Chart loading
// is protected by per-Source mutexes to ensure thread-safe lazy initialization.
type Renderer struct {
	settings   *cli.EnvSettings
	inputs     []*sourceHolder
	helmEngine engine.Engine
	opts       RendererOptions
}

// New creates a new Helm Renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	rendererOpts := RendererOptions{
		Filters:      make([]types.Filter, 0),
		Transformers: make([]types.Transformer, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt.ApplyTo(&rendererOpts)
	}

	settings := rendererOpts.Settings
	if settings == nil {
		settings = cli.New()
	}

	// Wrap sources in holders and validate
	holders := make([]*sourceHolder, len(inputs))
	for i := range inputs {
		holders[i] = &sourceHolder{
			Source: inputs[i],
			mu:     &sync.RWMutex{},
		}
		if err := holders[i].Validate(); err != nil {
			return nil, err
		}
	}

	r := &Renderer{
		settings: settings,
		inputs:   holders,
		helmEngine: engine.Engine{
			LintMode: rendererOpts.LintMode,
			Strict:   rendererOpts.Strict,
		},
		opts: rendererOpts,
	}

	return r, nil
}

// Process executes the rendering logic for all configured inputs.
// It implements the types.Renderer interface.
// This method is safe for concurrent use.
func (r *Renderer) Process(ctx context.Context, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for i := range r.inputs {
		objects, err := r.renderSingle(ctx, r.inputs[i], renderTimeValues)
		if err != nil {
			return nil, fmt.Errorf(
				"error rendering helm chart %s (release: %s): %w",
				r.inputs[i].Chart,
				r.inputs[i].ReleaseName,
				err,
			)
		}

		// Apply renderer-level filters and transformers per-source for better error context
		transformed, err := pipeline.Apply(ctx, objects, r.opts.Filters, r.opts.Transformers)
		if err != nil {
			return nil, fmt.Errorf(
				"error applying filters/transformers to helm chart %s (release: %s): %w",
				r.inputs[i].Chart,
				r.inputs[i].ReleaseName,
				err,
			)
		}

		allObjects = append(allObjects, transformed...)
	}

	return allObjects, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

func (r *Renderer) values(ctx context.Context, holder *sourceHolder, renderTimeValues map[string]any) (map[string]any, error) {
	sourceValues := map[string]any{}

	if holder.Values != nil {
		v, err := holder.Values(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get values for chart %q (release %q): %w", holder.Chart, holder.ReleaseName, err)
		}
		sourceValues = v
	}

	// Deep merge with render-time values taking precedence
	return util.DeepMerge(sourceValues, renderTimeValues), nil
}

// prepareRenderValues gets values from the Values function, processes dependencies,
// and prepares render values using chartutil.ToRenderValues.
func (r *Renderer) prepareRenderValues(ctx context.Context, holder *sourceHolder, renderTimeValues map[string]any) (chartutil.Values, error) {
	// Get values dynamically (includes render-time values)
	values, err := r.values(ctx, holder, renderTimeValues)
	if err != nil {
		return nil, fmt.Errorf("failed to get values for chart %q (release %q): %w", holder.Chart, holder.ReleaseName, err)
	}

	// Process dependencies if enabled
	if holder.ProcessDependencies {
		if err := chartutil.ProcessDependencies(holder.chart, values); err != nil {
			return nil, fmt.Errorf("failed to process dependencies for chart %q (release %q): %w", holder.Chart, holder.ReleaseName, err)
		}
	}

	// Prepare render values
	renderValues, err := chartutil.ToRenderValues(
		holder.chart,
		values,
		chartutil.ReleaseOptions{
			Name:      holder.ReleaseName,
			Revision:  1,
			IsInstall: true,
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare render values for chart %q (release %q): %w", holder.Chart, holder.ReleaseName, err)
	}

	return renderValues, nil
}

// renderSingle performs the rendering for a single Helm chart.
// It processes dependencies, prepares render values, renders the templates,
// and converts the output to unstructured objects.
func (r *Renderer) renderSingle(ctx context.Context, holder *sourceHolder, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
	// Load chart if not already loaded (thread-safe lazy loading)
	chart, err := holder.LoadChart(r.settings)
	if err != nil {
		return nil, err
	}

	// Prepare render values (includes render-time values)
	renderValues, err := r.prepareRenderValues(ctx, holder, renderTimeValues)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare render values for chart %q (release %q): %w", holder.Chart, holder.ReleaseName, err)
	}

	// Compute cache key from chart identifier and render values
	type cacheKeyData struct {
		Chart          string
		ReleaseName    string
		ReleaseVersion string
		RenderValues   chartutil.Values
	}

	var cacheKey string

	// Check cache (if enabled)
	if r.opts.Cache != nil {
		cacheKey = dump.ForHash(cacheKeyData{
			Chart:          holder.Chart,
			ReleaseName:    holder.ReleaseName,
			ReleaseVersion: holder.ReleaseVersion,
			RenderValues:   renderValues,
		})

		// ensure objects are evicted
		r.opts.Cache.Sync()

		if cached, found := r.opts.Cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Render the chart
	files, err := r.helmEngine.Render(chart, renderValues)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart %q (release %q): %w", holder.Chart, holder.ReleaseName, err)
	}

	result := make([]unstructured.Unstructured, 0)

	// Process CRDs first
	crdObjects, err := r.processCRDs(chart, holder)
	if err != nil {
		return nil, err
	}
	result = append(result, crdObjects...)

	// Process rendered templates
	templateObjects, err := r.processRenderedTemplates(files, holder)
	if err != nil {
		return nil, err
	}
	result = append(result, templateObjects...)

	// Cache result (if enabled)
	if r.opts.Cache != nil {
		r.opts.Cache.Set(cacheKey, result)
	}

	return result, nil
}
