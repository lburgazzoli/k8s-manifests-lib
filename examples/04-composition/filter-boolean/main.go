package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Filter Boolean Composition Example ===\n")
	fmt.Println("Demonstrates: filter.And(), filter.Or(), filter.Not()")
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

	// Complex filter using boolean composition:
	// - Exclude system namespaces (Not)
	// - Keep only Deployments OR Services (Or)
	f := filter.And(
		// Exclude system namespaces
		filter.Not(
			namespace.Filter("kube-system", "kube-public", "kube-node-lease"),
		),
		// Only keep Deployments and Services
		filter.Or(
			gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
			gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
		),
	)

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects (Deployments and Services, excluding system namespaces)\n", len(objects))

	// Show another example: production Deployments with specific labels OR staging Services
	fmt.Println("\n=== Example 2: Complex OR Logic ===\n")

	complexFilter := filter.Or(
		filter.And(
			namespace.Filter("production"),
			gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
			labels.HasLabel("critical"),
		),
		filter.And(
			namespace.Filter("staging"),
			gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
		),
	)

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(complexFilter),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects (production critical Deployments OR staging Services)\n", len(objects2))
}
