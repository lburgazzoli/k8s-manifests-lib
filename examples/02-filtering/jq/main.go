package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/jq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	jqutil "github.com/lburgazzoli/k8s-manifests-lib/pkg/util/jq"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== JQ Filtering Example ===")
	l.Log("Demonstrates: Filtering objects using JQ expressions")
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

	// Example 1: Filter by API version
	l.Log("1. API Version - Keep only apps/v1 resources")
	appsV1Filter, err := jq.Filter(`.apiVersion == "apps/v1"`)
	if err != nil {
		return fmt.Errorf("failed to create filter: %w", err)
	}

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(appsV1Filter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects1, err := e1.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d apps/v1 objects\n\n", len(objects1))

	// Example 2: Complex boolean expression
	l.Log("2. Boolean Logic - Keep Deployments OR Services")
	orFilter, err := jq.Filter(`.kind == "Deployment" or .kind == "Service"`)
	if err != nil {
		return fmt.Errorf("failed to create filter: %w", err)
	}

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(orFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects2, err := e2.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d objects (Deployments or Services)\n\n", len(objects2))

	// Example 3: JQ with variables
	l.Log("3. With Variables - Filter by dynamic kind")
	varFilter, err := jq.Filter(
		`.kind == $expectedKind`,
		jqutil.WithVariable("expectedKind", "Deployment"),
	)
	if err != nil {
		return fmt.Errorf("failed to create filter: %w", err)
	}

	e3, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(varFilter),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects3, err := e3.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("   Rendered %d Deployments (using JQ variable)\n", len(objects3))

	return nil
}
