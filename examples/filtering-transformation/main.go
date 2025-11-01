package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/jq"
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
	l := logger.FromContext(ctx)
	l.Log("=== Filtering & Transformation Pipeline ===")
	l.Log("Demonstrates: Render → Filter → Transform workflow")
	l.Log("")

	// Create a Helm renderer
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
			Values: helm.Values(map[string]any{
				"replicaCount": 3,
			}),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Create a JQ filter to keep only Deployments
	deploymentFilter, err := jq.Filter(`.kind == "Deployment"`)
	if err != nil {
		return fmt.Errorf("failed to create filter: %w", err)
	}

	// Create a label transformer to add common labels
	labelTransformer := labels.Set(map[string]string{
		"env":                          "production",
		"tier":                         "frontend",
		"app.kubernetes.io/managed-by": "k8s-manifests-lib",
	})

	// Create engine with filter and transformer
	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(deploymentFilter),      // Applied first
		engine.WithTransformer(labelTransformer), // Applied to filtered objects
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Render
	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Print results
	l.Logf("Rendered %d objects after filtering and transformation\n\n", len(objects))

	// Show the pipeline in action
	l.Log("Pipeline steps:")
	l.Log("  1. Render: Helm chart → multiple objects")
	l.Log("  2. Filter: Keep only Deployments (JQ: `.kind == \"Deployment\"`)")
	l.Log("  3. Transform: Add production labels to all filtered objects")
	l.Log("")

	// Show results
	for i, obj := range objects {
		l.Logf("%d. %s/%s\n", i+1, obj.GetKind(), obj.GetName())
		l.Logf("   Labels: %v\n\n", obj.GetLabels())
	}

	return nil
}
