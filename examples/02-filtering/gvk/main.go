package main

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
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
	l.Log("=== GVK (Group/Version/Kind) Filtering Example ===")
	l.Log("Demonstrates: Filtering objects by Kind and API version")
	l.Log("")

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Example 1: Filter by single Kind
	l.Log("1. Single Kind - Keep only Deployments")
	deploymentFilter := gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment"))

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(deploymentFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects1, err := e1.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d Deployment objects\n\n", len(objects1))

	// Example 2: Filter by multiple Kinds
	l.Log("2. Multiple Kinds - Keep Deployments and Services")
	multiKindFilter := gvk.Filter(
		appsv1.SchemeGroupVersion.WithKind("Deployment"),
		corev1.SchemeGroupVersion.WithKind("Service"),
	)

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(multiKindFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects (Deployments and Services)\n", len(objects2))

	// Show what kinds were rendered
	kindCounts := make(map[string]int)
	for _, obj := range objects2 {
		kindCounts[obj.GetKind()]++
	}
	for kind, count := range kindCounts {
		l.Logf("   - %d %s(s)\n", count, kind)
	}

	return nil
}
