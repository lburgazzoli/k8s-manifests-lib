package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Filter Conditional Composition Example ===")
	fmt.Println("Demonstrates: filter.If() for conditional filtering")
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

	// Conditional filter: only apply label filter if in production namespace
	// If object is in production, it must have "critical" label to pass
	// If object is NOT in production, it passes without label check
	f := filter.If(
		namespace.Filter("production"),
		labels.HasLabel("critical"),
	)

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects\n", len(objects))
	fmt.Println("(Production objects must have 'critical' label, others pass through)")

	// Show another example: combine multiple conditional filters
	fmt.Println("\n=== Example 2: Multiple Conditional Filters ===")

	multiFilter := filter.And(
		// All objects must be Deployments
		gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
		// If in production, must have "critical" label
		filter.If(
			namespace.Filter("production"),
			labels.HasLabel("critical"),
		),
		// If in staging, must have "tested" label
		filter.If(
			namespace.Filter("staging"),
			labels.HasLabel("tested"),
		),
	)

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(multiFilter),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d Deployments with environment-specific label requirements\n", len(objects2))
}
