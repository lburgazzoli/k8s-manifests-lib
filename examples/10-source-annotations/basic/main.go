package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

//go:embed manifests/*.yaml
var manifestsFS embed.FS

func main() {
	fmt.Println("=== Source Annotations Example ===")
	fmt.Println("Demonstrates: Tracking the source of rendered objects with automatic annotations")
	fmt.Println()

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
		log.Fatalf("Failed to create Helm renderer: %v", err)
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
		log.Fatalf("Failed to create YAML renderer: %v", err)
	}

	// Create engine with renderers
	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithRenderer(yamlRenderer),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Render with source annotations
	fmt.Println("=== Rendering with Source Annotations ===")
	objects, err := e.Render(ctx)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	// Display objects with their source annotations
	fmt.Printf("Rendered %d objects with source tracking:\n\n", len(objects))
	for i, obj := range objects {
		fmt.Printf("%d. %s/%s (namespace: %s)\n", i+1, obj.GetKind(), obj.GetName(), obj.GetNamespace())

		annotations := obj.GetAnnotations()
		if annotations != nil {
			fmt.Println("   Source Annotations:")
			if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
				fmt.Printf("   - Type: %s\n", sourceType)
			}
			if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
				fmt.Printf("   - Path: %s\n", sourcePath)
			}
			if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
				fmt.Printf("   - File: %s\n", sourceFile)
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Use Cases ===")
	fmt.Println("✓ Track which renderer produced each object")
	fmt.Println("✓ Debug multi-source configurations")
	fmt.Println("✓ Audit and compliance tracking")
	fmt.Println("✓ Filter or process objects based on source")
	fmt.Println("✓ Understand object provenance in complex pipelines")
}
