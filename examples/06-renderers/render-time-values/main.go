package main

import (
	"context"
	"fmt"
	"log"

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
	l := logger.FromContext(ctx)
	l.Log("=== Render-Time Values Example ===")
	l.Log("Demonstrates: Overriding configured values at render-time with deep merging")
	l.Log()

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
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	e, err := engine.New(engine.WithRenderer(helmRenderer))
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// First render: Use configured values
	l.Log("=== Render 1: Using Configured Values ===")
	objects1, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	l.Logf("Rendered %d objects with configured values (replicaCount=2, tag=1.25.0)\n", len(objects1))

	// Second render: Override values at render-time
	// Render-time values are deep merged with configured values
	// Only the specified keys are overridden; others remain unchanged
	l.Log("\n=== Render 2: Overriding Values at Render-Time ===")
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
		return fmt.Errorf("failed to render: %w", err)
	}
	l.Logf("Rendered %d objects with overridden values (replicaCount=5, tag=1.26.0)\n", len(objects2))

	// Third render: Different overrides
	l.Log("\n=== Render 3: Different Overrides ===")
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
		return fmt.Errorf("failed to render: %w", err)
	}
	l.Logf("Rendered %d objects with different overrides (replicaCount=10, service.type=LoadBalancer)\n", len(objects3))

	// Fourth render: Back to configured values (no overrides)
	l.Log("\n=== Render 4: Back to Configured Values ===")
	objects4, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	l.Logf("Rendered %d objects with configured values again (replicaCount=2, tag=1.25.0)\n", len(objects4))

	l.Log("\n=== Key Points ===")
	l.Log("✓ Render-time values override configured values for each Render() call")
	l.Log("✓ Deep merging: Only specified keys are overridden, others remain unchanged")
	l.Log("✓ Each render is independent - overrides don't affect subsequent renders")
	l.Log("✓ Useful for: environment-specific configs, testing, dynamic parameters")

	return nil
}
