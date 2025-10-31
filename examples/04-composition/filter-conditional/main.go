package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
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
	log := logger.FromContext(ctx)
	log.Log("=== Filter Conditional Composition Example ===")
	log.Log("Demonstrates: filter.If() for conditional filtering")
	log.Log()

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

	// Conditional filter: only apply label filter if in production namespace
	// If object is in production, it must have "critical" label to pass
	// If object is NOT in production, it passes without label check
	f := filter.If(
		namespace.Filter("production"),
		labels.HasLabel("critical"),
	)

	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("Rendered %d objects\n", len(objects))
	log.Log("(Production objects must have 'critical' label, others pass through)")

	// Show another example: combine multiple conditional filters
	log.Log("\n=== Example 2: Multiple Conditional Filters ===")

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

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(multiFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("Rendered %d Deployments with environment-specific label requirements\n", len(objects2))

	return nil
}
