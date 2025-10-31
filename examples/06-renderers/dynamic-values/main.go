package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rs/xid"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Dynamic Values Example ===")
	log.Log("Demonstrates: Using dynamic values function for runtime configuration")
	log.Log()

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

				log.Log("Fetching dynamic configuration...")

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
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	e, err := engine.New(engine.WithRenderer(helmRenderer))
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// First render
	log.Log("\n=== First Render ===")
	objects1, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	log.Logf("Rendered %d objects with dynamic values\n", len(objects1))

	// Wait a bit
	time.Sleep(1 * time.Second)

	// Second render - values function is called again
	log.Log("\n=== Second Render ===")
	objects2, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	log.Logf("Rendered %d objects with new dynamic values\n", len(objects2))
	log.Log("\nNote: Each render calls the values function, allowing for fresh configuration")

	return nil
}
