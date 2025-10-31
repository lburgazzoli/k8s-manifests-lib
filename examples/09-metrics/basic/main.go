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
	l := logger.FromContext(ctx)
	l.Log("=== Rendering Metrics Example ===")
	l.Log("Demonstrates: Collecting render performance metrics")
	l.Log()

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
	l.Logf("Rendering %d times...\n\n", iterations)

	for i := range iterations {
		objects, err := e.Render(ctx)
		if err != nil {
			return fmt.Errorf("failed to render iteration %d: %w", i+1, err)
		}
		l.Logf("Render %d: Rendered %d objects\n", i+1, len(objects))
	}

	// Print metrics summary
	l.Logf("\n=== Metrics Summary ===\n\n")

	// Render-level metrics
	renderSummary := renderMetric.Summary()
	l.Log("Render Metrics:")
	l.Logf("  Total Renders: %d\n", renderSummary.TotalRenders)
	l.Logf("  Average Duration: %v\n", renderSummary.AverageDuration)
	l.Logf("  Total Objects: %d\n", renderSummary.TotalObjects)

	// Renderer-specific metrics
	rendererSummary := rendererMetric.Summary()
	l.Log("\nRenderer Metrics:")
	for name, stats := range rendererSummary {
		l.Logf("  %s:\n", name)
		l.Logf("    Executions: %d\n", stats.Executions)
		l.Logf("    Avg Duration: %v\n", stats.AverageDuration)
		l.Logf("    Total Objects: %d\n", stats.TotalObjects)
		l.Logf("    Errors: %d\n", stats.Errors)
	}

	l.Log("\n=== Notes ===")
	l.Log("- Metrics are optional and have zero overhead when not used")
	l.Log("- Each metric type has its own interface (client-go pattern)")
	l.Log("- Easy to integrate with Prometheus, OpenTelemetry, etc.")

	return nil
}
