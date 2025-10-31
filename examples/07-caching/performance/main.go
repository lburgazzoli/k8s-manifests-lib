package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
)

func benchmarkRender(ctx context.Context, name string, e *engine.Engine, iterations int) error {
	log := logger.FromContext(ctx)
	log.Logf("\n=== %s ===\n", name)

	var totalDuration time.Duration

	for i := range iterations {
		start := time.Now()
		objects, err := e.Render(ctx)
		if err != nil {
			return fmt.Errorf("failed to render iteration %d: %w", i+1, err)
		}
		duration := time.Since(start)
		totalDuration += duration

		log.Logf("  Iteration %d: %v (%d objects)\n", i+1, duration, len(objects))
	}

	avgDuration := totalDuration / time.Duration(iterations)
	log.Logf("  Average: %v\n", avgDuration)

	return nil
}

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Cache Performance Comparison ===")
	log.Log("Demonstrates: Performance benefits of caching")
	log.Log("")

	iterations := 3

	// Without cache
	log.Log("Creating renderer WITHOUT cache...")
	noCacheRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "no-cache-nginx",
			Values: helm.Values(map[string]any{
				"replicaCount": 3,
			}),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create no-cache renderer: %w", err)
	}

	noCacheEngine, err := engine.New(engine.WithRenderer(noCacheRenderer))
	if err != nil {
		return fmt.Errorf("failed to create no-cache engine: %w", err)
	}
	if err := benchmarkRender(ctx, "Without Cache", noCacheEngine, iterations); err != nil {
		return err
	}

	// With cache
	log.Log("\nCreating renderer WITH cache...")
	cacheRenderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
				ReleaseName: "cache-nginx",
				Values: helm.Values(map[string]any{
					"replicaCount": 3,
				}),
			},
		},
		helm.WithCache(cache.WithTTL(5*time.Minute)),
	)
	if err != nil {
		return fmt.Errorf("failed to create cache renderer: %w", err)
	}

	cacheEngine, err := engine.New(engine.WithRenderer(cacheRenderer))
	if err != nil {
		return fmt.Errorf("failed to create cache engine: %w", err)
	}
	if err := benchmarkRender(ctx, "With Cache", cacheEngine, iterations); err != nil {
		return err
	}

	log.Log("\n=== Summary ===")
	log.Log("Without cache: Every render fetches from source (slow)")
	log.Log("With cache: First render fetches, subsequent renders use cache (fast)")
	log.Log("Cache automatically deep clones results to prevent pollution")

	return nil
}
