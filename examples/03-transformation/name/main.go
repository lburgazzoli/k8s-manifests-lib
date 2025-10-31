package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Name Transformation Example ===")
	l.Log("Demonstrates: Modifying object names with prefix, suffix, and replace")
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

	// Example 1: Add prefix to names
	l.Log("1. SetPrefix - Add 'prod-' prefix to all object names")
	prefixTransformer := name.SetPrefix("prod-")

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(prefixTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects1, err := e1.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Transformed %d objects (added 'prod-' prefix)\n", len(objects1))
	if len(objects1) > 0 {
		l.Logf("   Example: %s\n\n", objects1[0].GetName())
	}

	// Example 2: Add suffix to names
	l.Log("2. SetSuffix - Add '-v2' suffix to all object names")
	suffixTransformer := name.SetSuffix("-v2")

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(suffixTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Transformed %d objects (added '-v2' suffix)\n", len(objects2))
	if len(objects2) > 0 {
		l.Logf("   Example: %s\n\n", objects2[0].GetName())
	}

	// Example 3: Replace substring in names
	l.Log("3. Replace - Replace 'staging' with 'production' in names")
	replaceTransformer := name.Replace("staging", "production")

	e3, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(replaceTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects3, err := e3.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Transformed %d objects (replaced 'staging' → 'production')\n", len(objects3))
	if len(objects3) > 0 {
		l.Logf("   Example: %s\n", objects3[0].GetName())
	}

	return nil
}
