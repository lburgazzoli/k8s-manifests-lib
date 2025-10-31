package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

func main() {
	ctx := context.Background()

	// Create three YAML renderers that will process different manifest directories
	yamlRenderer1, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("manifests1"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create YAML renderer 1: %v", err)
	}

	yamlRenderer2, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("manifests2"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create YAML renderer 2: %v", err)
	}

	yamlRenderer3, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("manifests3"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create YAML renderer 3: %v", err)
	}

	// Sequential rendering (default)
	fmt.Println("=== Sequential Rendering ===")
	sequentialEngine, err := engine.New(
		engine.WithRenderer(yamlRenderer1),
		engine.WithRenderer(yamlRenderer2),
		engine.WithRenderer(yamlRenderer3),
	)
	if err != nil {
		log.Fatal(err)
	}

	start := time.Now()
	objects, err := sequentialEngine.Render(ctx)
	if err != nil {
		log.Fatalf("Sequential render failed: %v", err)
	}
	sequentialTime := time.Since(start)
	fmt.Printf("Rendered %d objects in %v\n\n", len(objects), sequentialTime)

	// Parallel rendering
	fmt.Println("=== Parallel Rendering ===")
	parallelEngine, err := engine.New(
		engine.WithRenderer(yamlRenderer1),
		engine.WithRenderer(yamlRenderer2),
		engine.WithRenderer(yamlRenderer3),
		engine.WithParallel(true), // Enable parallel rendering
	)
	if err != nil {
		log.Fatal(err)
	}

	start = time.Now()
	objects, err = parallelEngine.Render(ctx)
	if err != nil {
		log.Fatalf("Parallel render failed: %v", err)
	}
	parallelTime := time.Since(start)
	fmt.Printf("Rendered %d objects in %v\n\n", len(objects), parallelTime)

	// Using struct-based options
	fmt.Println("=== Struct-based Options ===")
	structEngine, err := engine.New(&engine.EngineOptions{
		Renderers: []types.Renderer{
			yamlRenderer1,
			yamlRenderer2,
			yamlRenderer3,
		},
		Parallel: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	start = time.Now()
	objects, err = structEngine.Render(ctx)
	if err != nil {
		log.Fatalf("Struct-based render failed: %v", err)
	}
	structTime := time.Since(start)
	fmt.Printf("Rendered %d objects in %v\n", len(objects), structTime)

	// Print speedup
	if parallelTime < sequentialTime {
		speedup := float64(sequentialTime) / float64(parallelTime)
		fmt.Printf("\nSpeedup: %.2fx faster\n", speedup)
	}
}
