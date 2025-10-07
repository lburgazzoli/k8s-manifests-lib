//nolint:unused
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
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	labelstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
	nstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

// Example 1: Multi-Environment Deployment Pipeline.
func multiEnvironmentPipeline() {
	fmt.Println("=== Example 1: Multi-Environment Deployment Pipeline ===")

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

	fmt.Printf("Rendered %d objects\n\n", len(objects))
}

// Example 2: Selective Resource Processing.
func selectiveResourceProcessing() {
	fmt.Println("=== Example 2: Selective Resource Processing ===")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Complex filter: production Deployments with specific labels OR staging Services
	f := filter.Or(
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

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects matching complex filter\n\n", len(objects))
}

// Example 3: Conditional Transformations.
func conditionalTransformations() {
	fmt.Println("=== Example 3: Conditional Transformations ===")

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

	fmt.Printf("Rendered %d objects with conditional transformations\n\n", len(objects))
}

// Example 4: Label and Annotation Cleanup.
func labelCleanup() {
	fmt.Println("=== Example 4: Label and Annotation Cleanup ===")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Remove temporary labels and annotations
	t := transformer.Chain(
		// Remove specific temporary labels
		labelstrans.Remove("temp", "debug", "test-only"),

		// Remove all labels with 'temp-' prefix
		labelstrans.RemoveIf(func(key string, value string) bool {
			return len(key) > 5 && key[:5] == "temp-"
		}),

		// Remove annotations with specific values
		annotations.RemoveIf(func(key string, value string) bool {
			return value == "delete-me" || value == "temporary"
		}),

		// Add production labels
		labelstrans.Set(map[string]string{
			"env":  "production",
			"tier": "frontend",
		}),
	)

	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(t),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Rendered %d objects with cleanup transformations\n\n", len(objects))
}

// Example 5: Name-Based Filtering and Transformation.
func nameBasedProcessing() {
	fmt.Println("=== Example 5: Name-Based Filtering and Transformation ===")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Filter by name patterns
	f := filter.Or(
		filter.And(
			namespace.Filter("production"),
			// Only resources with 'critical' prefix in production
			filter.Or(
				gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
				gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
			),
		),
	)

	// Add environment-specific name prefix
	t := transformer.Switch(
		[]transformer.Case{
			{
				When: namespace.Filter("production"),
				Then: nametrans.SetPrefix("prod-"),
			},
			{
				When: namespace.Filter("staging"),
				Then: nametrans.SetPrefix("stg-"),
			},
		},
		nametrans.SetPrefix("dev-"),
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

	fmt.Printf("Rendered %d objects with name-based processing\n\n", len(objects))
}

// Example 6: Complex Nested Composition.
func complexNestedComposition() {
	fmt.Println("=== Example 6: Complex Nested Composition ===")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry.example.com/myapp:1.0.0",
			ReleaseName: "myapp",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Deeply nested filter logic
	f := filter.And(
		filter.Not(
			namespace.Filter("kube-system", "kube-public"),
		),
		filter.Or(
			filter.And(
				namespace.Filter("production"),
				filter.Or(
					gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
					gvk.Filter(appsv1.SchemeGroupVersion.WithKind("StatefulSet")),
				),
				labels.HasLabel("critical"),
			),
			filter.And(
				namespace.Filter("staging", "development"),
				gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
			),
		),
	)

	// Nested transformer composition
	t := transformer.Chain(
		nstrans.EnsureDefault("default"),
		transformer.Switch(
			[]transformer.Case{
				{
					When: namespace.Filter("production"),
					Then: transformer.Chain(
						labelstrans.Set(map[string]string{
							"env":        "prod",
							"tier":       "critical",
							"monitoring": "enabled",
						}),
						annotations.Set(map[string]string{
							"sla":            "99.99",
							"alert-severity": "critical",
						}),
						transformer.If(
							gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
							labelstrans.Set(map[string]string{
								"deployment-strategy": "rolling",
							}),
						),
					),
				},
			},
			labelstrans.Set(map[string]string{"env": "dev"}),
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

	fmt.Printf("Rendered %d objects with complex nested composition\n\n", len(objects))
}

// Uncomment to run this example standalone
// func main() {
// 	// Run all examples
// 	multiEnvironmentPipeline()
// 	selectiveResourceProcessing()
// 	conditionalTransformations()
// 	labelCleanup()
// 	nameBasedProcessing()
// 	complexNestedComposition()
// }
