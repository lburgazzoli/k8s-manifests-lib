package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	labelstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Complex Nested Composition Example ===")
	l.Log("Demonstrates: Deep nesting of filters and transformers")
	l.Log()

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "myapp",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Deeply nested filter logic:
	// - Exclude system namespaces
	// - Include either:
	//   - Production Deployments/StatefulSets with "critical" label
	//   - Staging/dev Services
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

	// Nested transformer composition:
	// - Ensure default namespace
	// - Switch on namespace with nested conditionals
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
						// Nested conditional within the production case
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

	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
		engine.WithTransformer(t),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Rendered %d objects with complex nested filtering and transformations\n\n", len(objects))

	l.Log("Filter Logic:")
	l.Log("  Exclude: kube-system, kube-public")
	l.Log("  Include:")
	l.Log("    - Production Deployments/StatefulSets with 'critical' label")
	l.Log("    - Staging/dev Services")
	l.Log()
	l.Log("Transformer Logic:")
	l.Log("  Always: Ensure default namespace")
	l.Log("  Production:")
	l.Log("    - Add critical labels + SLA annotations")
	l.Log("    - If Deployment: Add deployment-strategy label")
	l.Log("  Default: Add dev environment label")

	// Show examples
	for i, obj := range objects {
		if i >= 3 {
			break
		}
		l.Logf("\n%d. %s/%s (namespace: %s)\n", i+1, obj.GetKind(), obj.GetName(), obj.GetNamespace())
		l.Logf("   Labels: %v\n", obj.GetLabels())
		if len(obj.GetAnnotations()) > 0 {
			l.Logf("   Annotations: %v\n", obj.GetAnnotations())
		}
	}

	return nil
}
