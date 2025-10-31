package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/memory"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	log := logger.FromContext(ctx)
	log.Log("=== Rendering Metrics Example ===")
	log.Log("Demonstrates: Collecting render performance metrics")
	log.Log()

	// Create metrics collectors
	renderMetric := &memory.RenderMetric{}
	rendererMetric := memory.NewRendererMetric()

	// Attach to context
	ctx = metrics.WithMetrics(ctx, &metrics.Metrics{
		RenderMetric:   renderMetric,
		RendererMetric: rendererMetric,
	})

	// Create engine with Helm renderer
	e, err := engine.Helm(
		helm.Source{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
			Values:      helm.Values(map[string]any{"replicaCount": 3}),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Perform multiple renders
	iterations := 3
	log.Logf("Rendering %d times...\n\n", iterations)

	for i := range iterations {
		objects, err := e.Render(ctx)
		if err != nil {
			return fmt.Errorf("failed to render iteration %d: %w", i+1, err)
		}
		log.Logf("Render %d: Rendered %d objects\n", i+1, len(objects))
	}

	// Print metrics summary
	log.Logf("\n=== Metrics Summary ===\n\n")

	// Render-level metrics
	renderSummary := renderMetric.Summary()
	log.Log("Render Metrics:")
	log.Logf("  Total Renders: %d\n", renderSummary.TotalRenders)
	log.Logf("  Average Duration: %v\n", renderSummary.AverageDuration)
	log.Logf("  Total Objects: %d\n", renderSummary.TotalObjects)

	// Renderer-specific metrics
	rendererSummary := rendererMetric.Summary()
	log.Log("\nRenderer Metrics:")
	for name, stats := range rendererSummary {
		log.Logf("  %s:\n", name)
		log.Logf("    Executions: %d\n", stats.Executions)
		log.Logf("    Avg Duration: %v\n", stats.AverageDuration)
		log.Logf("    Total Objects: %d\n", stats.TotalObjects)
		log.Logf("    Errors: %d\n", stats.Errors)
	}

	log.Log("\n=== Notes ===")
	log.Log("- Metrics are optional and have zero overhead when not used")
	log.Log("- Each metric type has its own interface (client-go pattern)")
	log.Log("- Easy to integrate with Prometheus, OpenTelemetry, etc.")

	return nil
}
