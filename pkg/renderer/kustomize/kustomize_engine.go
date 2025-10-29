package kustomize

import (
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

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize/unionfs"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

type Engine struct {
	kustomizer *krusty.Kustomizer
	fs         filesys.FileSystem
	opts       *RendererOptions
}

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
		return nil, fmt.Errorf("unable to read kustomization: %w", err)
	}

	fs := e.fs
	addedOriginAnnotations := false

	if e.opts.SourceAnnotations || len(values) > 0 {
		p, f, err := e.fs.CleanedAbs(input.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %q: %w", input.Path, err)
		}
		if f != "" {
			return nil, fmt.Errorf("path %q must be a dir: %w", input.Path, err)
		}

		builder := unionfs.NewBuilder(e.fs)

		// Add modified kustomization if source annotations are enabled
		if e.opts.SourceAnnotations {
			if !slices.Contains(kust.BuildMetadata, kustomizetypes.OriginAnnotations) {
				kust.BuildMetadata = append(kust.BuildMetadata, kustomizetypes.OriginAnnotations)
				addedOriginAnnotations = true

				data, err := goyaml.Marshal(kust)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal kustomization: %w", err)
				}

				builder.WithOverride(filepath.Join(p.String(), name), data)
			}
		}

		// Add values ConfigMap if provided
		if len(values) > 0 {
			valuesContent, err := createValuesConfigMapYAML(values)
			if err != nil {
				return nil, fmt.Errorf("failed to create values ConfigMap: %w", err)
			}
			builder.WithOverride(filepath.Join(p.String(), "values.yaml"), valuesContent)
		}

		fs, err = builder.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to create union filesystem: %w", err)
		}
	}

	resMap, err := kustomizer.Run(fs, input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize: %w", err)
	}

	for _, t := range e.opts.Plugins {
		if err := t.Transform(resMap); err != nil {
			return nil, fmt.Errorf("failed to apply kustomize plugin transformer: %w", err)
		}
	}

	return e.toUnstructured(resMap, input.Path, addedOriginAnnotations)
}

func (e *Engine) toUnstructured(resMap resmap.ResMap, path string, removeConfig bool) ([]unstructured.Unstructured, error) {
	result := make([]unstructured.Unstructured, resMap.Size())

	for i, res := range resMap.Resources() {
		m, err := res.Map()
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource to map: %w", err)
		}

		result[i] = unstructured.Unstructured{}

		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, &result[i]); err != nil {
			return nil, fmt.Errorf("failed to convert map to unstructured: %w", err)
		}

		if e.opts.SourceAnnotations {
			annotations := result[i].GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}

			annotations[types.AnnotationSourceType] = rendererType
			annotations[types.AnnotationSourcePath] = path

			if origin, err := res.GetOrigin(); err == nil && origin != nil {
				annotations[types.AnnotationSourceFile] = origin.Path
			}

			// Remove config.kubernetes.io/origin if we added OriginAnnotations ourselves
			if removeConfig {
				delete(annotations, "config.kubernetes.io/origin")
			}

			result[i].SetAnnotations(annotations)
		}
	}

	return result, nil
}
