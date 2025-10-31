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
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Filter Boolean Composition Example ===")
	l.Log("Demonstrates: filter.And(), filter.Or(), filter.Not()")
	l.Log()

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

	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(f),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Rendered %d objects (Deployments and Services, excluding system namespaces)\n", len(objects))

	// Show another example: production Deployments with specific labels OR staging Services
	l.Log("\n=== Example 2: Complex OR Logic ===")

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

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(complexFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Rendered %d objects (production critical Deployments OR staging Services)\n", len(objects2))

	return nil
}
