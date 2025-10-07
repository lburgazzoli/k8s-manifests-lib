package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
)

func benchmarkRender(name string, e *engine.Engine, iterations int) {
	fmt.Printf("\n=== %s ===\n", name)

	ctx := context.Background()
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		objects, err := e.Render(ctx)
		if err != nil {
			log.Fatal(err)
		}
		duration := time.Since(start)
		totalDuration += duration

		fmt.Printf("  Iteration %d: %v (%d objects)\n", i+1, duration, len(objects))
	}

	avgDuration := totalDuration / time.Duration(iterations)
	fmt.Printf("  Average: %v\n", avgDuration)
}

func main() {
	fmt.Println("=== Cache Performance Comparison ===\n")
	fmt.Println("Demonstrates: Performance benefits of caching")
	fmt.Println()

	iterations := 3

	// Without cache
	fmt.Println("Creating renderer WITHOUT cache...")
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
		log.Fatal(err)
	}

	noCacheEngine := engine.New(engine.WithRenderer(noCacheRenderer))
	benchmarkRender("Without Cache", noCacheEngine, iterations)

	// With cache
	fmt.Println("\nCreating renderer WITH cache...")
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
		helm.WithCache(cache.WithTTL(5 * time.Minute)),
	)
	if err != nil {
		log.Fatal(err)
	}

	cacheEngine := engine.New(engine.WithRenderer(cacheRenderer))
	benchmarkRender("With Cache", cacheEngine, iterations)

	fmt.Println("\n=== Summary ===")
	fmt.Println("Without cache: Every render fetches from source (slow)")
	fmt.Println("With cache: First render fetches, subsequent renders use cache (fast)")
	fmt.Println("Cache automatically deep clones results to prevent pollution")
}
