package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
)

func main() {
	fmt.Println("=== Basic Kustomize Example ===\n")
	fmt.Println("Demonstrates: Simple Kustomize directory rendering using engine.Kustomize() convenience function")
	fmt.Println()

	// Create an Engine with a single Kustomize renderer
	// Point to a directory containing a kustomization.yaml file
	e, err := engine.Kustomize(kustomize.Source{
		Path: "./kustomization-example", // Path to kustomization directory
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
	fmt.Printf("Successfully rendered %d Kubernetes objects from Kustomize\n\n", len(objects))

	// Show what was rendered
	fmt.Println("Rendered objects:")
	for i, obj := range objects {
		fmt.Printf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			fmt.Printf(" (namespace: %s)", obj.GetNamespace())
		}
		fmt.Println()
	}
}
