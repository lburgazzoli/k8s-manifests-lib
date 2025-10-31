package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	// Create three YAML renderers that will process different manifest directories
	yamlRenderer1, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("manifests1"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create yaml renderer 1: %w", err)
	}

	yamlRenderer2, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("manifests2"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create yaml renderer 2: %w", err)
	}

	yamlRenderer3, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("manifests3"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create yaml renderer 3: %w", err)
	}

	// Sequential rendering (default)
	log.Log("=== Sequential Rendering ===")
	sequentialEngine, err := engine.New(
		engine.WithRenderer(yamlRenderer1),
		engine.WithRenderer(yamlRenderer2),
		engine.WithRenderer(yamlRenderer3),
	)
	if err != nil {
		return fmt.Errorf("failed to create sequential engine: %w", err)
	}

	start := time.Now()
	objects, err := sequentialEngine.Render(ctx)
	if err != nil {
		return fmt.Errorf("sequential render failed: %w", err)
	}
	sequentialTime := time.Since(start)
	log.Logf("Rendered %d objects in %v\n\n", len(objects), sequentialTime)

	// Parallel rendering
	log.Log("=== Parallel Rendering ===")
	parallelEngine, err := engine.New(
		engine.WithRenderer(yamlRenderer1),
		engine.WithRenderer(yamlRenderer2),
		engine.WithRenderer(yamlRenderer3),
		engine.WithParallel(true), // Enable parallel rendering
	)
	if err != nil {
		return fmt.Errorf("failed to create parallel engine: %w", err)
	}

	start = time.Now()
	objects, err = parallelEngine.Render(ctx)
	if err != nil {
		return fmt.Errorf("parallel render failed: %w", err)
	}
	parallelTime := time.Since(start)
	log.Logf("Rendered %d objects in %v\n\n", len(objects), parallelTime)

	// Using struct-based options
	log.Log("=== Struct-based Options ===")
	structEngine, err := engine.New(&engine.EngineOptions{
		Renderers: []types.Renderer{
			yamlRenderer1,
			yamlRenderer2,
			yamlRenderer3,
		},
		Parallel: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create struct-based engine: %w", err)
	}

	start = time.Now()
	objects, err = structEngine.Render(ctx)
	if err != nil {
		return fmt.Errorf("struct-based render failed: %w", err)
	}
	structTime := time.Since(start)
	log.Logf("Rendered %d objects in %v\n", len(objects), structTime)

	// Print speedup
	if parallelTime < sequentialTime {
		speedup := float64(sequentialTime) / float64(parallelTime)
		log.Logf("\nSpeedup: %.2fx faster\n", speedup)
	}

	return nil
}
