package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
	nstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Transformer Chain Composition Example ===")
	l.Log("Demonstrates: transformer.Chain() for sequential transformations")
	l.Log()

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "myapp",
			Values: helm.Values(map[string]any{
				"replicaCount": 3,
			}),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Chain multiple transformers to be applied in sequence
	t := transformer.Chain(
		// 1. Ensure default namespace if not set
		nstrans.EnsureDefault("default"),

		// 2. Add managed-by label to all resources
		labels.Set(map[string]string{
			"app.kubernetes.io/managed-by": "k8s-manifests-lib",
		}),

		// 3. Add production labels
		labels.Set(map[string]string{
			"env":  "production",
			"tier": "frontend",
		}),

		// 4. Add annotations
		annotations.Set(map[string]string{
			"deployed-by": "example-script",
			"version":     "1.0.0",
		}),

		// 5. Add name prefix
		nametrans.SetPrefix("prod-"),
	)

	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(t),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Rendered %d objects with chained transformations:\n", len(objects))
	l.Log("  1. Default namespace ensured")
	l.Log("  2. Managed-by label added")
	l.Log("  3. Environment labels added")
	l.Log("  4. Annotations added")
	l.Log("  5. Name prefix 'prod-' added")

	// Example object to show the transformations
	if len(objects) > 0 {
		obj := objects[0]
		l.Logf("\nFirst object: %s/%s\n", obj.GetKind(), obj.GetName())
		l.Logf("Namespace: %s\n", obj.GetNamespace())
		l.Logf("Labels: %v\n", obj.GetLabels())
		l.Logf("Annotations: %v\n", obj.GetAnnotations())
	}

	return nil
}
