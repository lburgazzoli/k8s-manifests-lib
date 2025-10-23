package kustomize

import (
	"fmt"
	"slices"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

type Engine struct {
	kustomizer        *krusty.Kustomizer
	fs                filesys.FileSystem
	sourceAnnotations bool
	plugins           []resmap.Transformer
}

func NewEngine(fs filesys.FileSystem, sourceAnnotations bool, plugins []resmap.Transformer) *Engine {
	return &Engine{
		kustomizer: krusty.MakeKustomizer(&krusty.Options{
			LoadRestrictions: kustomizetypes.LoadRestrictionsRootOnly,
			PluginConfig:     &kustomizetypes.PluginConfig{},
		}),
		fs:                fs,
		sourceAnnotations: sourceAnnotations,
		plugins:           plugins,
	}
}

func (e *Engine) Run(path string) ([]unstructured.Unstructured, error) {
	kust, err := readKustomization(e.fs, path)
	if err != nil {
		return nil, fmt.Errorf("unable to read kustomization: %w", err)
	}

	fs := e.fs
	addedOriginAnnotations := false

	if e.sourceAnnotations {
		if !slices.Contains(kust.BuildMetadata, kustomizetypes.OriginAnnotations) {
			kust.BuildMetadata = append(kust.BuildMetadata, kustomizetypes.OriginAnnotations)
			addedOriginAnnotations = true
		}

		ofs, err := newOverrideFS(e.fs, path, kust)
		if err != nil {
			return nil, err
		}

		fs = ofs
	}

	resMap, err := e.kustomizer.Run(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize: %w", err)
	}

	for _, t := range e.plugins {
		if err := t.Transform(resMap); err != nil {
			return nil, fmt.Errorf("failed to apply kustomize plugin transformer: %w", err)
		}
	}

	return e.toUnstructured(resMap, path, addedOriginAnnotations)
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

		if e.sourceAnnotations {
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
