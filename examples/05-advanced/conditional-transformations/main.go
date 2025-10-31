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
	l.Log("=== Conditional Transformations Example ===")
	l.Log("Demonstrates: transformer.If() for applying transformations conditionally")
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

	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(t),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Rendered %d objects with conditional transformations:\n", len(objects))
	l.Log("  1. Default namespace ensured (always)")
	l.Log("  2. Managed-by label added (always)")
	l.Log("  3. Monitoring labels added (only if production namespace)")
	l.Log("  4. Cost-center annotation added (only for Deployments and StatefulSets)")

	// Show examples of rendered objects
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
