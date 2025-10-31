package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rs/xid"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Multiple Helm Sources Example ===")
	log.Log("Demonstrates: Rendering multiple Helm charts with a single renderer")
	log.Log()

	// Create a Helm renderer with multiple source charts
	// Each chart is processed independently and results are aggregated
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
			ReleaseName: "app-one",
			Values: helm.Values(map[string]any{
				"shared": map[string]any{
					"appId": xid.New().String(),
				},
			}),
		},
		{
			Repo:        "https://dapr.github.io/helm-charts",
			Chart:       "dapr",
			ReleaseName: "app-two",
			Values: helm.Values(map[string]any{
				"shared": map[string]any{
					"appId": xid.New().String(),
				},
			}),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	e, err := engine.New(engine.WithRenderer(helmRenderer))
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	log.Logf("Successfully rendered %d objects from %d Helm charts\n\n", len(objects), 2)

	// Count objects per release
	releaseCounts := make(map[string]int)
	for _, obj := range objects {
		labels := obj.GetLabels()
		if release, ok := labels["app.kubernetes.io/instance"]; ok {
			releaseCounts[release]++
		}
	}

	log.Log("Objects per release:")
	for release, count := range releaseCounts {
		log.Logf("  - %s: %d objects\n", release, count)
	}

	return nil
}
