package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
)

//go:embed manifests/*.yaml
var manifestsFS embed.FS

func main() {
	fmt.Println("=== Basic YAML Example ===")
	fmt.Println("Demonstrates: Simple YAML file loading using engine.Yaml() convenience function")
	fmt.Println()

	// Create an Engine with a single YAML renderer
	// Using embedded filesystem for portability
	e, err := engine.Yaml(yaml.Source{
		FS:   manifestsFS,
		Path: "manifests/*.yaml", // Glob pattern to match YAML files
	})
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Render the manifests
	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	// Print summary
	fmt.Printf("Successfully loaded %d Kubernetes objects from YAML files\n\n", len(objects))

	// Show what was loaded
	fmt.Println("Loaded objects:")
	for i, obj := range objects {
		fmt.Printf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			fmt.Printf(" (namespace: %s)", obj.GetNamespace())
		}
		fmt.Println()
	}
}
