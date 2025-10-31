package kustomize

import (
	"context"
	"fmt"

	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/dump"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

const rendererType = "kustomize"

// Source represents the input for a Kustomize rendering operation.
type Source struct {
	// Path specifies the directory containing kustomization.yaml.
	// Must be a valid filesystem path to a kustomization root.
	Path string

	// Values provides dynamic key-value data written as a ConfigMap.
	// Function is called during rendering to obtain dynamic values.
	// The values are written to a ConfigMap file at Path/values.yaml.
	//
	// IMPORTANT: Values are NOT applied automatically to resources.
	// The kustomization must explicitly use this ConfigMap via:
	// - replacements: to substitute values in resources
	// - configMapGenerator: if integrating with generated configs
	// - patches: to modify resources based on values
	//
	// If Path/values.yaml already exists, rendering will fail with an error
	// to prevent accidental overwrites.
	Values func(context.Context) (map[string]string, error)

	// LoadRestrictions specifies restrictions on what can be referenced.
	// If LoadRestrictionsUnknown (zero value), uses the renderer-wide default.
	// Set to LoadRestrictionsRootOnly or LoadRestrictionsNone to override.
	LoadRestrictions kustomizetypes.LoadRestrictions
}

// Renderer is a renderer that uses kustomize to render resources.
type Renderer struct {
	inputs []*sourceHolder
	fs     filesys.FileSystem
	engine *Engine
	opts   *RendererOptions
}

// New creates a new kustomize renderer.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	// Initialize renderer options
	rendererOpts := RendererOptions{
		Filters:          make([]types.Filter, 0),
		Transformers:     make([]types.Transformer, 0),
		Plugins:          make([]resmap.Transformer, 0),
		LoadRestrictions: kustomizetypes.LoadRestrictionsRootOnly,
	}

	// Apply all options to RendererOptions
	for _, opt := range opts {
		opt.ApplyTo(&rendererOpts)
	}

	// Wrap sources in holders and validate
	holders := make([]*sourceHolder, len(inputs))
	for i := range inputs {
		holders[i] = &sourceHolder{
			Source: inputs[i],
		}
		if err := holders[i].Validate(); err != nil {
			return nil, err
		}
	}

	fs := filesys.MakeFsOnDisk()
	r := &Renderer{
		inputs: holders,
		fs:     fs,
		engine: NewEngine(fs, &rendererOpts),
		opts:   &rendererOpts,
	}

	return r, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

// Process implements types.Renderer by rendering the kustomize resources and applying filters and transformers.
func (r *Renderer) Process(ctx context.Context, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for _, holder := range r.inputs {
		objects, err := r.renderSingle(ctx, holder, renderTimeValues)
		if err != nil {
			return nil, fmt.Errorf("error rendering kustomize path %s: %w", holder.Path, err)
		}

		// Apply renderer-level filters and transformers per-source for better error context
		transformed, err := pipeline.Apply(ctx, objects, r.opts.Filters, r.opts.Transformers)
		if err != nil {
			return nil, fmt.Errorf(
				"error applying filters/transformers to path %s: %w",
				holder.Path,
				err,
			)
		}

		allObjects = append(allObjects, transformed...)
	}

	return allObjects, nil
}

// renderSingle performs the rendering for a single kustomize path.
func (r *Renderer) renderSingle(ctx context.Context, holder *sourceHolder, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
	// Get values dynamically (includes render-time values)
	values, err := computeValues(ctx, holder.Source, renderTimeValues)
	if err != nil {
		return nil, fmt.Errorf("failed to get values for path %q: %w", holder.Path, err)
	}

	// Compute cache key from input Path and Values
	type cacheKeyData struct {
		Path   string
		Values map[string]string
	}

	var cacheKey string

	// Check cache (if enabled)
	if r.opts.Cache != nil {
		cacheKey = dump.ForHash(cacheKeyData{
			Path:   holder.Path,
			Values: values,
		})

		// ensure objects are evicted
		r.opts.Cache.Sync()

		if cached, found := r.opts.Cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// No filesystem writes needed - values passed to engine
	result, err := r.engine.Run(holder.Source, values)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize for path %q: %w", holder.Path, err)
	}

	// Cache result (if enabled)
	if r.opts.Cache != nil {
		r.opts.Cache.Set(cacheKey, result)
	}

	return result, nil
}
