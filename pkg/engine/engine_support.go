package engine

import (
	"fmt"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/mem"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
)

// Helm creates an Engine configured with a single Helm renderer.
// This is a convenience function for simple Helm-only rendering scenarios.
//
// Example:
//
//	e, _ := engine.Helm(
//	    helm.Source{
//	        Chart:       "oci://registry/chart:1.0.0",
//	        ReleaseName: "my-release",
//	        Values:      helm.Values(map[string]any{"replicas": 3}),
//	    },
//	    helm.WithCache(cache.WithTTL(5*time.Minute)),
//	)
//	objects, _ := e.Render(ctx)
func Helm(source helm.Source, opts ...helm.RendererOption) (*Engine, error) {
	renderer, err := helm.New([]helm.Source{source}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create helm renderer: %w", err)
	}
	return New(WithRenderer(renderer))
}

// Kustomize creates an Engine configured with a single Kustomize renderer.
// This is a convenience function for simple Kustomize-only rendering scenarios.
//
// Example:
//
//	e, _ := engine.Kustomize(kustomize.Source{
//	    Path: "/path/to/kustomization",
//	})
//	objects, _ := e.Render(ctx)
func Kustomize(source kustomize.Source, opts ...kustomize.RendererOption) (*Engine, error) {
	renderer, err := kustomize.New([]kustomize.Source{source}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kustomize renderer: %w", err)
	}
	return New(WithRenderer(renderer))
}

// Yaml creates an Engine configured with a single YAML renderer.
// This is a convenience function for simple YAML-only rendering scenarios.
//
// Example:
//
//	e, _ := engine.Yaml(yaml.Source{
//	    FS:   os.DirFS("/path/to/manifests"),
//	    Path: "*.yaml",
//	})
//	objects, _ := e.Render(ctx)
func Yaml(source yaml.Source, opts ...yaml.RendererOption) (*Engine, error) {
	renderer, err := yaml.New([]yaml.Source{source}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create yaml renderer: %w", err)
	}
	return New(WithRenderer(renderer))
}

// GoTemplate creates an Engine configured with a single Go template renderer.
// This is a convenience function for simple Go template-only rendering scenarios.
//
// Example:
//
//	e, _ := engine.GoTemplate(gotemplate.Source{
//	    FS:   os.DirFS("/path/to/templates"),
//	    Path: "*.yaml.tmpl",
//	})
//	objects, _ := e.Render(ctx)
func GoTemplate(source gotemplate.Source, opts ...gotemplate.RendererOption) (*Engine, error) {
	sources := []gotemplate.Source{source}
	renderer, err := gotemplate.New(sources, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gotemplate renderer: %w", err)
	}
	return New(WithRenderer(renderer))
}

// Mem creates an Engine configured with a single memory renderer.
// This is a convenience function for simple in-memory rendering scenarios.
//
// Example:
//
//	e, _ := engine.Mem(mem.Source{
//	    Objects: []unstructured.Unstructured{...},
//	})
//	objects, _ := e.Render(ctx)
func Mem(source mem.Source, opts ...mem.RendererOption) (*Engine, error) {
	renderer, err := mem.New([]mem.Source{source}, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create mem renderer: %w", err)
	}
	return New(WithRenderer(renderer))
}
