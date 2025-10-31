package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

//go:embed manifests/*.yaml
var manifestsFS embed.FS

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Source Annotations Example ===")
	l.Log("Demonstrates: Tracking the source of rendered objects with automatic annotations")
	l.Log()

	// Create multiple renderers with source annotations enabled
	// Source annotations are enabled at the renderer level
	helmRenderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
				ReleaseName: "my-nginx",
				Values:      helm.Values(map[string]any{"replicaCount": 1}),
			},
		},
		helm.WithSourceAnnotations(true), // Enable source tracking for Helm
	)
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	yamlRenderer, err := yaml.New(
		[]yaml.Source{
			{
				FS:   manifestsFS,
				Path: "manifests/*.yaml",
			},
		},
		yaml.WithSourceAnnotations(true), // Enable source tracking for YAML
	)
	if err != nil {
		return fmt.Errorf("failed to create yaml renderer: %w", err)
	}

	// Create engine with renderers
	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithRenderer(yamlRenderer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Render with source annotations
	l.Log("=== Rendering with Source Annotations ===")
	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Display objects with their source annotations
	l.Logf("Rendered %d objects with source tracking:\n\n", len(objects))
	for i, obj := range objects {
		l.Logf("%d. %s/%s (namespace: %s)\n", i+1, obj.GetKind(), obj.GetName(), obj.GetNamespace())

		annotations := obj.GetAnnotations()
		if annotations != nil {
			l.Log("   Source Annotations:")
			if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
				l.Logf("   - Type: %s\n", sourceType)
			}
			if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
				l.Logf("   - Path: %s\n", sourcePath)
			}
			if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
				l.Logf("   - File: %s\n", sourceFile)
			}
		}
		l.Log()
	}

	l.Log("=== Use Cases ===")
	l.Log("✓ Track which renderer produced each object")
	l.Log("✓ Debug multi-source configurations")
	l.Log("✓ Audit and compliance tracking")
	l.Log("✓ Filter or process objects based on source")
	l.Log("✓ Understand object provenance in complex pipelines")

	return nil
}
