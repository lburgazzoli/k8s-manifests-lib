package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Label Transformation Example ===")
	log.Log("Demonstrates: Adding, updating, and removing labels")
	log.Log("")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Example 1: Add/Update labels
	log.Log("1. Set - Add or update labels")
	setTransformer := labels.Set(map[string]string{
		"env":                          "production",
		"tier":                         "frontend",
		"app.kubernetes.io/managed-by": "k8s-manifests-lib",
		"app.kubernetes.io/part-of":    "nginx-stack",
	})

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(setTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects1, err := e1.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("   Transformed %d objects (added 4 labels)\n", len(objects1))
	if len(objects1) > 0 {
		log.Logf("   Example labels: %v\n\n", objects1[0].GetLabels())
	}

	// Example 2: Remove specific labels
	log.Log("2. Remove - Remove specific label keys")
	removeTransformer := labels.Remove("temp", "debug", "test-only")

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(removeTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("   Transformed %d objects (removed temp/debug/test-only labels)\n\n", len(objects2))

	// Example 3: Remove labels conditionally
	log.Log("3. RemoveIf - Remove labels matching a condition")
	removeIfTransformer := labels.RemoveIf(func(key string, value string) bool {
		// Remove all labels with 'temp-' prefix
		return strings.HasPrefix(key, "temp-")
	})

	e3, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(removeIfTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects3, err := e3.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("   Transformed %d objects (removed labels with 'temp-' prefix)\n", len(objects3))

	return nil
}
