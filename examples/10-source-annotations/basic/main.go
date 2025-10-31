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
	log := logger.FromContext(ctx)
	log.Log("=== Source Annotations Example ===")
	log.Log("Demonstrates: Tracking the source of rendered objects with automatic annotations")
	log.Log()

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
	log.Log("=== Rendering with Source Annotations ===")
	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Display objects with their source annotations
	log.Logf("Rendered %d objects with source tracking:\n\n", len(objects))
	for i, obj := range objects {
		log.Logf("%d. %s/%s (namespace: %s)\n", i+1, obj.GetKind(), obj.GetName(), obj.GetNamespace())

		annotations := obj.GetAnnotations()
		if annotations != nil {
			log.Log("   Source Annotations:")
			if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
				log.Logf("   - Type: %s\n", sourceType)
			}
			if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
				log.Logf("   - Path: %s\n", sourcePath)
			}
			if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
				log.Logf("   - File: %s\n", sourceFile)
			}
		}
		log.Log()
	}

	log.Log("=== Use Cases ===")
	log.Log("✓ Track which renderer produced each object")
	log.Log("✓ Debug multi-source configurations")
	log.Log("✓ Audit and compliance tracking")
	log.Log("✓ Filter or process objects based on source")
	log.Log("✓ Understand object provenance in complex pipelines")

	return nil
}
