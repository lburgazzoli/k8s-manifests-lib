package kustomize

import (
	"fmt"
	"path/filepath"
	"slices"

	goyaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kresource "sigs.k8s.io/kustomize/api/resource"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize/unionfs"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

type (
	resMap   = resmap.ResMap
	resource = *kresource.Resource
)

// Engine wraps a Kustomize kustomizer for rendering kustomization directories.
type Engine struct {
	kustomizer *krusty.Kustomizer
	fs         filesys.FileSystem
	opts       *RendererOptions
}

// NewEngine creates a new kustomize rendering engine.
func NewEngine(fs filesys.FileSystem, opts *RendererOptions) *Engine {
	return &Engine{
		kustomizer: krusty.MakeKustomizer(&krusty.Options{
			LoadRestrictions: opts.LoadRestrictions,
			PluginConfig:     &kustomizetypes.PluginConfig{},
		}),
		fs:   fs,
		opts: opts,
	}
}

// Run executes the kustomize build process for the given source and returns the rendered objects.
func (e *Engine) Run(input Source, values map[string]string) ([]unstructured.Unstructured, error) {
	restrictions := e.opts.LoadRestrictions
	if input.LoadRestrictions != kustomizetypes.LoadRestrictionsUnknown {
		restrictions = input.LoadRestrictions
	}

	// Create kustomizer with appropriate restrictions
	kustomizer := krusty.MakeKustomizer(&krusty.Options{
		LoadRestrictions: restrictions,
		PluginConfig:     &kustomizetypes.PluginConfig{},
	})

	kust, name, err := readKustomization(e.fs, input.Path)
	if err != nil {
		return nil, fmt.Errorf("unable to read kustomization from path %q: %w", input.Path, err)
	}

	// Prepare filesystem with overlays if needed
	fs, addedOriginAnnotations, err := e.prepareFilesystem(input.Path, kust, name, values)
	if err != nil {
		return nil, err
	}

	resMap, err := kustomizer.Run(fs, input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize for path %q: %w", input.Path, err)
	}

	for _, t := range e.opts.Plugins {
		if err := t.Transform(resMap); err != nil {
			return nil, fmt.Errorf("failed to apply kustomize plugin transformer for path %q: %w", input.Path, err)
		}
	}

	// Convert ResMap to unstructured objects
	result, err := e.convertResources(resMap, input.Path)
	if err != nil {
		return nil, err
	}

	// Remove config.kubernetes.io/origin if we added OriginAnnotations ourselves
	if addedOriginAnnotations {
		for i := range result {
			removeOriginAnnotation(&result[i])
		}
	}

	return result, nil
}

// prepareFilesystem creates a union filesystem with overlays if needed for source annotations or values.
// Returns the filesystem to use, whether origin annotations were added, and any error.
func (e *Engine) prepareFilesystem(
	inputPath string,
	kust *kustomizetypes.Kustomization,
	kustName string,
	values map[string]string,
) (filesys.FileSystem, bool, error) {
	// If neither source annotations nor values are needed, use the base filesystem
	if !e.opts.SourceAnnotations && len(values) == 0 {
		return e.fs, false, nil
	}

	p, f, err := e.fs.CleanedAbs(inputPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to resolve path %q: %w", inputPath, err)
	}
	if f != "" {
		return nil, false, fmt.Errorf("path %q must be a dir: %w", inputPath, err)
	}

	builder := unionfs.NewBuilder(e.fs)
	addedOriginAnnotations := false

	// Add modified kustomization if source annotations are enabled
	if e.opts.SourceAnnotations {
		if !slices.Contains(kust.BuildMetadata, kustomizetypes.OriginAnnotations) {
			kust.BuildMetadata = append(kust.BuildMetadata, kustomizetypes.OriginAnnotations)
			addedOriginAnnotations = true

			data, err := goyaml.Marshal(kust)
			if err != nil {
				return nil, false, fmt.Errorf("failed to marshal kustomization: %w", err)
			}

			builder.WithOverride(filepath.Join(p.String(), kustName), data)
		}
	}

	// Add values ConfigMap if provided
	if len(values) > 0 {
		valuesContent, err := createValuesConfigMapYAML(values)
		if err != nil {
			return nil, false, fmt.Errorf("failed to create values ConfigMap: %w", err)
		}
		builder.WithOverride(filepath.Join(p.String(), "values.yaml"), valuesContent)
	}

	fs, err := builder.Build()
	if err != nil {
		return nil, false, fmt.Errorf("failed to create union filesystem: %w", err)
	}

	return fs, addedOriginAnnotations, nil
}

// addSourceAnnotationsToObject adds source tracking annotations to a single unstructured object.
// Only modifies the object if source annotations are enabled in engine options.
// Removes config.kubernetes.io/origin annotation if addedOriginAnnotations is true.
func (e *Engine) addSourceAnnotationsToObject(
	obj *unstructured.Unstructured,
	inputPath string,
	res resource,
) {
	if !e.opts.SourceAnnotations {
		return
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[types.AnnotationSourceType] = rendererType
	annotations[types.AnnotationSourcePath] = inputPath

	if origin, err := res.GetOrigin(); err == nil && origin != nil {
		annotations[types.AnnotationSourceFile] = origin.Path
	}

	obj.SetAnnotations(annotations)
}

// removeOriginAnnotation removes the config.kubernetes.io/origin annotation from an object.
// Used when we added OriginAnnotations ourselves to avoid duplication.
func removeOriginAnnotation(obj *unstructured.Unstructured) {
	annotations := obj.GetAnnotations()
	if annotations != nil {
		delete(annotations, "config.kubernetes.io/origin")
		obj.SetAnnotations(annotations)
	}
}

// convertResources converts a Kustomize ResMap to a slice of unstructured objects.
// Adds source annotations to each object if enabled.
func (e *Engine) convertResources(
	resMap resMap,
	inputPath string,
) ([]unstructured.Unstructured, error) {
	result := make([]unstructured.Unstructured, resMap.Size())

	for i, res := range resMap.Resources() {
		m, err := res.Map()
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource %s to map: %w", res.CurId(), err)
		}

		result[i] = unstructured.Unstructured{}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, &result[i]); err != nil {
			return nil, fmt.Errorf("failed to convert map to unstructured for resource %s: %w", res.CurId(), err)
		}

		e.addSourceAnnotationsToObject(&result[i], inputPath, res)
	}

	return result, nil
}
