package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/jq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
)

func main() {
	// Create a Helm renderer for a chart
	helmRenderer, err := helm.New([]helm.Data{
		{
			ChartSource: "oci://registry.example.com/my-chart:1.0.0", // or "/path/to/chart"
			ReleaseName: "my-release",
			Namespace:   "my-namespace",
			Values: map[string]any{
				"replicaCount": 3,
				"image": map[string]any{
					"repository": "nginx",
					"tag":        "latest",
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	// Create a JQ filter for namespace selection
	namespaceFilter, err := jq.Filter(`.metadata.namespace == "my-namespace"`)
	if err != nil {
		log.Fatalf("Failed to create namespace filter: %v", err)
	}

	// Create the engine with initial configuration
	e := engine.New(
		// Add the Helm renderer
		engine.WithRenderer(helmRenderer),
		// Add a filter to only keep resources in my-namespace
		engine.WithFilter(namespaceFilter),
		// Add a transformer to add a common label
		engine.WithTransformer(labels.Transform(map[string]string{
			"app.kubernetes.io/managed-by": "my-operator",
		})),
	)

	// Create a context
	ctx := context.Background()

	// Create a JQ filter for kind selection
	kindFilter, err := jq.Filter(`.kind == "Deployment"`)
	if err != nil {
		log.Fatalf("Failed to create kind filter: %v", err)
	}

	// Render with additional render-time options
	objects, err := e.Render(ctx,
		// Add a render-time filter to only keep Deployments
		engine.WithRenderFilter(kindFilter),
		// Add a render-time transformer to add an environment label
		engine.WithRenderTransformer(labels.Transform(map[string]string{
			"environment": "production",
		})),
	)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	// Print the results
	fmt.Printf("Rendered %d objects:\n", len(objects))
	for _, obj := range objects {
		fmt.Printf("- %s/%s (%s)\n", obj.GetKind(), obj.GetName(), obj.GetNamespace())
		fmt.Printf("  Labels: %v\n", obj.GetLabels())
	}
}
