package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
)

func main() {
	fmt.Println("=== Transformer Switch Composition Example ===")
	fmt.Println("Demonstrates: transformer.Switch() for multi-branch transformations")
	fmt.Println()

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
			Values: helm.Values(map[string]any{
				"replicaCount": 3,
			}),
		},
	})
	if err != nil {
		log.Fatal(err)
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

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(t),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects with environment-specific transformations:\n", len(objects))
	fmt.Println("  - Production: critical labels, SLA annotations, 'prod-' prefix")
	fmt.Println("  - Staging: monitoring labels, 'stg-' prefix")
	fmt.Println("  - Default: dev labels, 'dev-' prefix")

	// Show the first object as example
	if len(objects) > 0 {
		obj := objects[0]
		fmt.Printf("\nFirst object: %s/%s\n", obj.GetKind(), obj.GetName())
		fmt.Printf("Labels: %v\n", obj.GetLabels())
		fmt.Printf("Annotations: %v\n", obj.GetAnnotations())
	}
}
