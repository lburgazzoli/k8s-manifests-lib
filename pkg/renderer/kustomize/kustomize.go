package kustomize

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	goyaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/util/dump"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
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
}

// Values returns a Values function that always returns the provided static values.
// This is a convenience helper for the common case of non-dynamic values.
func Values(values map[string]string) func(context.Context) (map[string]string, error) {
	return func(_ context.Context) (map[string]string, error) {
		return values, nil
	}
}

// Renderer is a renderer that uses kustomize to render resources.
type Renderer struct {
	inputs        []Source
	kustomizeOpts krusty.Options
	kustomizer    *krusty.Kustomizer
	filters       []types.Filter
	transformers  []types.Transformer  // for post-processing
	plugins       []resmap.Transformer // for kustomize-native/plugin transformers
	fs            filesys.FileSystem
	decoder       runtime.Decoder
	cache         cache.Interface[[]unstructured.Unstructured]
}

// New creates a new kustomize renderer.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	// Validate inputs
	for i, input := range inputs {
		if input.Path == "" {
			return nil, fmt.Errorf("input[%d]: Path is required", i)
		}
	}

	r := &Renderer{
		inputs: slices.Clone(inputs),
		kustomizeOpts: krusty.Options{
			LoadRestrictions: kustomizetypes.LoadRestrictionsRootOnly,
			PluginConfig:     &kustomizetypes.PluginConfig{},
		},
		fs:           filesys.MakeFsOnDisk(),
		decoder:      yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme),
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
		plugins:      make([]resmap.Transformer, 0),
	}

	for _, opt := range opts {
		opt.ApplyTo(r)
	}

	r.kustomizer = krusty.MakeKustomizer(&r.kustomizeOpts)

	return r, nil
}

// Process implements types.Renderer by rendering the kustomize resources and applying filters and transformers.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for i, input := range r.inputs {
		objects, err := r.renderSingle(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error rendering kustomize[%d] path %s: %w", i, input.Path, err)
		}

		allObjects = append(allObjects, objects...)
	}

	transformed, err := pipeline.Apply(ctx, allObjects, r.filters, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("kustomize renderer: %w", err)
	}

	return transformed, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

// writeValuesConfigMap writes values as a ConfigMap YAML file using the renderer's filesystem.
// Returns the path to the written file, or empty string if no values.
func (r *Renderer) writeValuesConfigMap(path string, values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}

	valuesPath := filepath.Join(path, "values.yaml")

	// Check if file exists
	if r.fs.Exists(valuesPath) {
		return "", fmt.Errorf("values.yaml already exists at %s, refusing to overwrite", valuesPath)
	}

	// Create ConfigMap structure
	configMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]string{
			"name": "values",
		},
		"data": values,
	}

	data, err := goyaml.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values ConfigMap: %w", err)
	}

	if err := r.fs.WriteFile(valuesPath, data); err != nil {
		return "", fmt.Errorf("failed to write values.yaml: %w", err)
	}

	return valuesPath, nil
}

func (r *Renderer) values(ctx context.Context, input Source) (map[string]string, error) {
	if input.Values == nil {
		return map[string]string{}, nil
	}

	v, err := input.Values(ctx)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// renderSingle performs the rendering for a single kustomize path.
func (r *Renderer) renderSingle(ctx context.Context, input Source) ([]unstructured.Unstructured, error) {
	// Get values dynamically
	values, err := r.values(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get values: %w", err)
	}

	// Compute cache key from input Path and Values
	type cacheKeyData struct {
		Path   string
		Values map[string]string
	}

	var cacheKey string

	// Check cache (if enabled)
	if r.cache != nil {
		cacheKey = dump.ForHash(cacheKeyData{
			Path:   input.Path,
			Values: values,
		})

		// ensure objects are evicted
		r.cache.Sync()

		if cached, found := r.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Write values ConfigMap if provided
	valuesPath, err := r.writeValuesConfigMap(input.Path, values)
	if err != nil {
		return nil, err
	}

	// Clean up values.yaml file after rendering
	if valuesPath != "" {
		defer func() {
			_ = r.fs.RemoveAll(valuesPath)
		}()
	}

	resMap, err := r.kustomizer.Run(r.fs, input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize: %w", err)
	}

	// Apply kustomize-native/plugin transformers
	for _, t := range r.plugins {
		err := t.Transform(resMap)
		if err != nil {
			return nil, fmt.Errorf("failed to apply kustomize plugin transformer: %w", err)
		}
	}

	// Convert resources directly to unstructured objects
	renderedRes := resMap.Resources()

	result := make([]unstructured.Unstructured, len(renderedRes))

	for i, res := range renderedRes {
		m, err := res.Map()
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource to map: %w", err)
		}

		result[i] = unstructured.Unstructured{}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, &result[i]); err != nil {
			return nil, fmt.Errorf("failed to convert map to unstructured: %w", err)
		}
	}

	// Cache result (if enabled)
	if r.cache != nil {
		r.cache.Set(cacheKey, result)
	}

	return result, nil
}
