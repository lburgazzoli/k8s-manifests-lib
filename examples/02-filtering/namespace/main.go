package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
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
	l.Log("=== Namespace Filtering Example ===")
	l.Log("Demonstrates: Filtering objects by namespace")
	l.Log("")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Example 1: Include only specific namespaces
	l.Log("1. Include Filter - Keep only objects in 'production' and 'staging'")
	includeFilter := namespace.Filter("production", "staging")

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(includeFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects1, err := e1.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects from production/staging namespaces\n\n", len(objects1))

	// Example 2: Exclude specific namespaces
	l.Log("2. Exclude Filter - Exclude system namespaces")
	excludeFilter := namespace.Exclude("kube-system", "kube-public", "kube-node-lease")

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(excludeFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects (excluding system namespaces)\n", len(objects2))

	return nil
}
