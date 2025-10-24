package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Render-Time Values Example ===")
	fmt.Println("Demonstrates: Overriding configured values at render-time with deep merging")
	fmt.Println()

	// Create a Helm renderer with initial configuration values
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
			Values: helm.Values(map[string]any{
				"replicaCount": 2,
				"image": map[string]any{
					"repository": "nginx",
					"tag":        "1.25.0",
					"pullPolicy": "IfNotPresent",
				},
				"service": map[string]any{
					"type": "ClusterIP",
					"port": 80,
				},
			}),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	e := engine.New(engine.WithRenderer(helmRenderer))
	ctx := context.Background()

	// First render: Use configured values
	fmt.Println("=== Render 1: Using Configured Values ===")
	objects1, err := e.Render(ctx)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}
	fmt.Printf("Rendered %d objects with configured values (replicaCount=2, tag=1.25.0)\n", len(objects1))

	// Second render: Override values at render-time
	// Render-time values are deep merged with configured values
	// Only the specified keys are overridden; others remain unchanged
	fmt.Println("\n=== Render 2: Overriding Values at Render-Time ===")
	objects2, err := e.Render(ctx,
		engine.WithValues(map[string]any{
			"replicaCount": 5, // Override replicaCount
			"image": map[string]any{
				"tag": "1.26.0", // Override tag, keep repository and pullPolicy from config
			},
			// service.type and service.port remain ClusterIP and 80 from config
		}),
	)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}
	fmt.Printf("Rendered %d objects with overridden values (replicaCount=5, tag=1.26.0)\n", len(objects2))

	// Third render: Different overrides
	fmt.Println("\n=== Render 3: Different Overrides ===")
	objects3, err := e.Render(ctx,
		engine.WithValues(map[string]any{
			"replicaCount": 10,
			"service": map[string]any{
				"type": "LoadBalancer", // Override service type
				// service.port remains 80 from config
			},
		}),
	)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}
	fmt.Printf("Rendered %d objects with different overrides (replicaCount=10, service.type=LoadBalancer)\n", len(objects3))

	// Fourth render: Back to configured values (no overrides)
	fmt.Println("\n=== Render 4: Back to Configured Values ===")
	objects4, err := e.Render(ctx)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}
	fmt.Printf("Rendered %d objects with configured values again (replicaCount=2, tag=1.25.0)\n", len(objects4))

	fmt.Println("\n=== Key Points ===")
	fmt.Println("✓ Render-time values override configured values for each Render() call")
	fmt.Println("✓ Deep merging: Only specified keys are overridden, others remain unchanged")
	fmt.Println("✓ Each render is independent - overrides don't affect subsequent renders")
	fmt.Println("✓ Useful for: environment-specific configs, testing, dynamic parameters")
}
