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
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
	labelstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Multi-Environment Deployment Pipeline Example ===")
	log.Log("Demonstrates: Combined filtering and environment-specific transformations")
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

	log.Logf("Rendered %d objects (Deployments and Services, excluding system namespaces)\n", len(objects))
	log.Log("\nEnvironment-specific transformations applied:")
	log.Log("  Production: critical labels + SLA annotations + 'prod-' prefix")
	log.Log("  Staging: monitoring labels + 'stg-' prefix")
	log.Log("  Dev: basic labels + 'dev-' prefix")

	// Show examples of rendered objects
	for i, obj := range objects {
		if i >= 3 {
			break
		}
		log.Logf("\n%d. %s/%s\n", i+1, obj.GetKind(), obj.GetName())
		log.Logf("   Namespace: %s\n", obj.GetNamespace())
		log.Logf("   Labels: %v\n", obj.GetLabels())
		if len(obj.GetAnnotations()) > 0 {
			log.Logf("   Annotations: %v\n", obj.GetAnnotations())
		}
	}

	return nil
}
