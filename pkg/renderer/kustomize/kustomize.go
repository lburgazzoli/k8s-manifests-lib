package kustomize

import (
	"context"
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// Renderer is a renderer that uses kustomize to render resources.
type Renderer struct {
	basePath      string
	kustomizeOpts krusty.Options
	kustomizer    *krusty.Kustomizer
	filters       []types.Filter
	transformers  []types.Transformer  // for post-processing
	plugins       []resmap.Transformer // for kustomize-native/plugin transformers
	fs            filesys.FileSystem
	decoder       runtime.Decoder
}

// New creates a new kustomize renderer.
func New(basePath string, opts ...Option) *Renderer {
	r := &Renderer{
		basePath: basePath,
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
		opt(r)
	}

	r.kustomizer = krusty.MakeKustomizer(&r.kustomizeOpts)

	return r
}

// Process implements types.Renderer by rendering the kustomize resources and applying filters and transformers.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	resMap, err := r.kustomizer.Run(r.fs, r.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize: %w", err)
	}

	// Apply kustomize-native/plugin transformers
	for _, t := range r.plugins {
		if err := t.Transform(resMap); err != nil {
			return nil, fmt.Errorf("failed to apply kustomize plugin transformer: %w", err)
		}
	}

	// Convert resources directly to unstructured objects
	renderedRes := resMap.Resources()

	objects := make([]unstructured.Unstructured, len(renderedRes))
	for i, res := range renderedRes {
		m, err := res.Map()
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource to map: %w", err)
		}

		objects[i] = unstructured.Unstructured{}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, &objects[i]); err != nil {
			return nil, fmt.Errorf("failed to convert map to unstructured: %w", err)
		}
	}

	// Apply filters (these still work on unstructured objects)
	filtered, err := util.ApplyFilters(ctx, objects, r.filters)
	if err != nil {
		return nil, fmt.Errorf("failed to apply filters: %w", err)
	}

	// Apply post-processing transformers
	transformed, err := util.ApplyTransformers(ctx, filtered, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("failed to apply transformers: %w", err)
	}

	return transformed, nil
}
