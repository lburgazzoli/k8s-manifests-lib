package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	labelstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

func main() {
	fmt.Println("=== Conditional Transformations Example ===\n")
	fmt.Println("Demonstrates: transformer.If() for applying transformations conditionally")
	fmt.Println()

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Apply transformations only when specific conditions are met
	t := transformer.Chain(
		// Always ensure default namespace
		nstrans.EnsureDefault("default"),

		// Add managed-by label to all resources
		labelstrans.Set(map[string]string{
			"app.kubernetes.io/managed-by": "k8s-manifests-lib",
		}),

		// Conditionally add monitoring labels only to production
		transformer.If(
			namespace.Filter("production"),
			labelstrans.Set(map[string]string{
				"monitoring": "prometheus",
				"alerting":   "enabled",
			}),
		),

		// Conditionally add cost-center annotation only to specific kinds
		transformer.If(
			filter.Or(
				gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
				gvk.Filter(appsv1.SchemeGroupVersion.WithKind("StatefulSet")),
			),
			annotations.Set(map[string]string{
				"cost-center": "engineering",
			}),
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

	fmt.Printf("Rendered %d objects with conditional transformations:\n", len(objects))
	fmt.Println("  1. Default namespace ensured (always)")
	fmt.Println("  2. Managed-by label added (always)")
	fmt.Println("  3. Monitoring labels added (only if production namespace)")
	fmt.Println("  4. Cost-center annotation added (only for Deployments and StatefulSets)")

	// Show examples of rendered objects
	for i, obj := range objects {
		if i >= 3 {
			break
		}
		fmt.Printf("\n%d. %s/%s (namespace: %s)\n", i+1, obj.GetKind(), obj.GetName(), obj.GetNamespace())
		fmt.Printf("   Labels: %v\n", obj.GetLabels())
		if len(obj.GetAnnotations()) > 0 {
			fmt.Printf("   Annotations: %v\n", obj.GetAnnotations())
		}
	}
}
