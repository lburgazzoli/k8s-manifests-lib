package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
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
	l.Log("=== Label Filtering Example ===")
	l.Log("Demonstrates: Filtering objects by labels")
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

	// Example 1: Check if label exists
	l.Log("1. HasLabel - Keep objects with 'app' label")
	hasLabelFilter := labels.HasLabel("app")

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(hasLabelFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects1, err := e1.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects with 'app' label\n\n", len(objects1))

	// Example 2: Match specific label values
	l.Log("2. MatchLabels - Keep objects matching exact labels")
	matchFilter := labels.MatchLabels(map[string]string{
		"app":     "nginx",
		"version": "1.0",
	})

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(matchFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects with app=nginx AND version=1.0\n\n", len(objects2))

	// Example 3: Kubernetes label selector syntax
	l.Log("3. Selector - Use Kubernetes label selector syntax")
	selectorFilter, err := labels.Selector("app=nginx,tier in (frontend,backend)")
	if err != nil {
		return fmt.Errorf("failed to create selector: %w", err)
	}

	e3, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(selectorFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects3, err := e3.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects matching selector\n", len(objects3))

	return nil
}
