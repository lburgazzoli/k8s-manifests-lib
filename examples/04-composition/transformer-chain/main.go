package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
	nstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

func main() {
	fmt.Println("=== Transformer Chain Composition Example ===")
	fmt.Println("Demonstrates: transformer.Chain() for sequential transformations")
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

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(t),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects with chained transformations:\n", len(objects))
	fmt.Println("  1. Default namespace ensured")
	fmt.Println("  2. Managed-by label added")
	fmt.Println("  3. Environment labels added")
	fmt.Println("  4. Annotations added")
	fmt.Println("  5. Name prefix 'prod-' added")

	// Example object to show the transformations
	if len(objects) > 0 {
		obj := objects[0]
		fmt.Printf("\nFirst object: %s/%s\n", obj.GetKind(), obj.GetName())
		fmt.Printf("Namespace: %s\n", obj.GetNamespace())
		fmt.Printf("Labels: %v\n", obj.GetLabels())
		fmt.Printf("Annotations: %v\n", obj.GetAnnotations())
	}
}
