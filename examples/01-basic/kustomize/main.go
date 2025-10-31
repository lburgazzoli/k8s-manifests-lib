package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Basic Kustomize Example ===")
	l.Log("Demonstrates: Simple Kustomize directory rendering using engine.Kustomize() convenience function")
	l.Log("")

	// Create an Engine with a single Kustomize renderer
	// Point to a directory containing a kustomization.yaml file
	e, err := engine.Kustomize(kustomize.Source{
		Path: "./kustomization-example", // Path to kustomization directory
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
	l.Logf("Successfully rendered %d Kubernetes objects from Kustomize\n\n", len(objects))

	// Show what was rendered
	l.Log("Rendered objects:")
	for i, obj := range objects {
		l.Logf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			l.Logf(" (namespace: %s)", obj.GetNamespace())
		}
		l.Log("")
	}

	return nil
}
