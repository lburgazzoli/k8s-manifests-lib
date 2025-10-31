package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Transformer Switch Composition Example ===")
	l.Log("Demonstrates: transformer.Switch() for multi-branch transformations")
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

	// Switch applies different transformations based on namespace
	// First matching case wins
	t := transformer.Switch(
		[]transformer.Case{
			{
				// Production environment
				When: namespace.Filter("production"),
				Then: transformer.Chain(
					labels.Set(map[string]string{
						"env":        "prod",
						"monitoring": "enabled",
						"backup":     "enabled",
					}),
					annotations.Set(map[string]string{
						"alert-severity": "critical",
						"sla":            "99.99",
					}),
					nametrans.SetPrefix("prod-"),
				),
			},
			{
				// Staging environment
				When: namespace.Filter("staging"),
				Then: transformer.Chain(
					labels.Set(map[string]string{
						"env":        "staging",
						"monitoring": "enabled",
					}),
					nametrans.SetPrefix("stg-"),
				),
			},
		},
		// Default transformation for dev and other environments
		transformer.Chain(
			labels.Set(map[string]string{"env": "dev"}),
			nametrans.SetPrefix("dev-"),
		),
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

	l.Logf("Rendered %d objects with environment-specific transformations:\n", len(objects))
	l.Log("  - Production: critical labels, SLA annotations, 'prod-' prefix")
	l.Log("  - Staging: monitoring labels, 'stg-' prefix")
	l.Log("  - Default: dev labels, 'dev-' prefix")

	// Show the first object as example
	if len(objects) > 0 {
		obj := objects[0]
		l.Logf("\nFirst object: %s/%s\n", obj.GetKind(), obj.GetName())
		l.Logf("Labels: %v\n", obj.GetLabels())
		l.Logf("Annotations: %v\n", obj.GetAnnotations())
	}

	return nil
}
