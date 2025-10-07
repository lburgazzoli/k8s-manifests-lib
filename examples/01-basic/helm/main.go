package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Basic Helm Example ===\n")
	fmt.Println("Demonstrates: Simple Helm chart rendering using engine.Helm() convenience function")
	fmt.Println()

	// Create an Engine with a single Helm renderer
	// This is the simplest way to render a Helm chart
	e, err := engine.Helm(helm.Source{
		Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
		ReleaseName: "my-nginx",
		Values: helm.Values(map[string]any{
			"replicaCount": 2,
			"service": map[string]any{
				"type": "ClusterIP",
			},
		}),
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
	fmt.Printf("Successfully rendered %d Kubernetes objects from Helm chart\n\n", len(objects))

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
