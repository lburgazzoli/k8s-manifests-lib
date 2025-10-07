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
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	labelstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
)

func main() {
	fmt.Println("=== Multi-Environment Deployment Pipeline Example ===\n")
	fmt.Println("Demonstrates: Combined filtering and environment-specific transformations")
	fmt.Println()

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
			Values: helm.Values(map[string]any{
				"replicaCount": 3,
				"image": map[string]any{
					"repository": "myapp",
					"tag":        "latest",
				},
			}),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Filter: Exclude system namespaces, keep only Deployments and Services
	f := filter.And(
		filter.Not(
			namespace.Filter("kube-system", "kube-public", "kube-node-lease"),
		),
		filter.Or(
			gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
			gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
		),
	)

	// Transform: Apply environment-specific labels, annotations, and name prefixes
	t := transformer.Switch(
		[]transformer.Case{
			{
				When: namespace.Filter("production"),
				Then: transformer.Chain(
					labelstrans.Set(map[string]string{
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
				When: namespace.Filter("staging"),
				Then: transformer.Chain(
					labelstrans.Set(map[string]string{
						"env":        "staging",
						"monitoring": "enabled",
					}),
					nametrans.SetPrefix("stg-"),
				),
			},
		},
		// Default for dev environments
		transformer.Chain(
			labelstrans.Set(map[string]string{"env": "dev"}),
			nametrans.SetPrefix("dev-"),
		),
	)

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
		engine.WithTransformer(t),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects (Deployments and Services, excluding system namespaces)\n", len(objects))
	fmt.Println("\nEnvironment-specific transformations applied:")
	fmt.Println("  Production: critical labels + SLA annotations + 'prod-' prefix")
	fmt.Println("  Staging: monitoring labels + 'stg-' prefix")
	fmt.Println("  Dev: basic labels + 'dev-' prefix")

	// Show examples of rendered objects
	for i, obj := range objects {
		if i >= 3 {
			break
		}
		fmt.Printf("\n%d. %s/%s\n", i+1, obj.GetKind(), obj.GetName())
		fmt.Printf("   Namespace: %s\n", obj.GetNamespace())
		fmt.Printf("   Labels: %v\n", obj.GetLabels())
		if len(obj.GetAnnotations()) > 0 {
			fmt.Printf("   Annotations: %v\n", obj.GetAnnotations())
		}
	}
}
