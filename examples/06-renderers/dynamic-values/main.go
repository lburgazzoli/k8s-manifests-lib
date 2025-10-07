package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rs/xid"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Dynamic Values Example ===")
	fmt.Println("Demonstrates: Using dynamic values function for runtime configuration")
	fmt.Println()

	// Create a Helm renderer with dynamic values
	// The values function is called at render time, allowing for runtime configuration
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "dynamic-nginx",
			Values: func(ctx context.Context) (map[string]any, error) {
				// Simulate fetching configuration at runtime
				// This could fetch from:
				// - External API
				// - Database
				// - Configuration file
				// - Environment variables
				// - Vault/secrets manager

				fmt.Println("Fetching dynamic configuration...")

				return map[string]any{
					"replicaCount": 3,
					"image": map[string]any{
						"repository": "nginx",
						"tag":        "latest",
					},
					// Dynamic values computed at render time
					"dynamicConfig": map[string]any{
						"appId":     xid.New().String(),
						"timestamp": time.Now().Unix(),
						"rendered":  true,
					},
				}, nil
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	e := engine.New(engine.WithRenderer(helmRenderer))

	// First render
	fmt.Println("\n=== First Render ===")
	objects1, err := e.Render(context.Background())
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}
	fmt.Printf("Rendered %d objects with dynamic values\n", len(objects1))

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Second render - values function is called again
	fmt.Println("\n=== Second Render ===")
	objects2, err := e.Render(context.Background())
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}
	fmt.Printf("Rendered %d objects with new dynamic values\n", len(objects2))
	fmt.Println("\nNote: Each render calls the values function, allowing for fresh configuration")
}
