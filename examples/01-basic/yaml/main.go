package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
)

//go:embed manifests/*.yaml
var manifestsFS embed.FS

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Basic YAML Example ===")
	log.Log("Demonstrates: Simple YAML file loading using engine.Yaml() convenience function")
	log.Log("")

	// Create an Engine with a single YAML renderer
	// Using embedded filesystem for portability
	e, err := engine.Yaml(yaml.Source{
		FS:   manifestsFS,
		Path: "manifests/*.yaml", // Glob pattern to match YAML files
	})
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Render the manifests
	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Print summary
	log.Logf("Successfully loaded %d Kubernetes objects from YAML files\n\n", len(objects))

	// Show what was loaded
	log.Log("Loaded objects:")
	for i, obj := range objects {
		log.Logf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			log.Logf(" (namespace: %s)", obj.GetNamespace())
		}
		log.Log("")
	}

	return nil
}
