package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Namespace Transformation Example ===")
	l.Log("Demonstrates: Setting and ensuring namespaces on objects")
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

	// Example 1: Force namespace unconditionally
	l.Log("1. Set - Force namespace to 'production'")
	setTransformer := namespace.Set("production")

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

	l.Logf("   Transformed %d objects (namespace → production)\n", len(objects1))
	if len(objects1) > 0 {
		l.Logf("   Example: %s/%s now in '%s' namespace\n\n",
			objects1[0].GetKind(), objects1[0].GetName(), objects1[0].GetNamespace())
	}

	// Example 2: Set default namespace only if empty
	l.Log("2. EnsureDefault - Set namespace only if not already set")
	ensureTransformer := namespace.EnsureDefault("default")

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(ensureTransformer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Transformed %d objects (namespace → default if empty)\n", len(objects2))
	l.Log("   Objects with existing namespaces are not modified")
	l.Log("   Objects without namespace get 'default'")

	return nil
}
