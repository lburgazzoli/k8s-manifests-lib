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

func main() {
	fmt.Println("=== Basic Caching Example ===\n")
	fmt.Println("Demonstrates: Enabling cache with TTL for improved performance")
	fmt.Println()

	// Create a Helm renderer with caching enabled
	helmRenderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
				ReleaseName: "cached-nginx",
				Values: helm.Values(map[string]any{
					"replicaCount": 3,
				}),
			},
		},
		// Enable caching with 5-minute TTL
		helm.WithCache(cache.WithTTL(5 * time.Minute)),
	)
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	e := engine.New(engine.WithRenderer(helmRenderer))
	ctx := context.Background()

	// First render: cache miss
	fmt.Println("1. First render (cache miss - fetches from source)")
	start := time.Now()
	objects1, err := e.Render(ctx)
	if err != nil {
		log.Fatal(err)
	}
	duration1 := time.Since(start)
	fmt.Printf("   Rendered %d objects in %v\n\n", len(objects1), duration1)

	// Second render: cache hit
	fmt.Println("2. Second render (cache hit - returns cached results)")
	start = time.Now()
	objects2, err := e.Render(ctx)
	if err != nil {
		log.Fatal(err)
	}
	duration2 := time.Since(start)
	fmt.Printf("   Rendered %d objects in %v\n\n", len(objects2), duration2)

	// Modify the cached result - won't affect cache due to automatic deep cloning
	fmt.Println("3. Modifying returned objects (won't affect cache)")
	if len(objects2) > 0 {
		originalName := objects2[0].GetName()
		objects2[0].SetName("modified-name")
		fmt.Printf("   Changed name from '%s' to '%s'\n\n", originalName, objects2[0].GetName())
	}

	// Third render: still gets original cached values
	fmt.Println("4. Third render (cache still has original values)")
	start = time.Now()
	objects3, err := e.Render(ctx)
	if err != nil {
		log.Fatal(err)
	}
	duration3 := time.Since(start)
	fmt.Printf("   Rendered %d objects in %v\n", len(objects3), duration3)
	if len(objects3) > 0 {
		fmt.Printf("   First object name: '%s' (not modified)\n", objects3[0].GetName())
	}

	fmt.Printf("\n Performance improvement: ~%.1fx faster with cache\n",
		float64(duration1)/float64(duration2))
}
