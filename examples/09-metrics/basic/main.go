package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/memory"
)

func main() {
	fmt.Println("=== Rendering Metrics Example ===")
	fmt.Println("Demonstrates: Collecting render performance metrics")
	fmt.Println()

	// Create metrics collectors
	renderMetric := &memory.RenderMetric{}
	rendererMetric := memory.NewRendererMetric()

	// Attach to context
	ctx := metrics.WithMetrics(context.Background(), &metrics.Metrics{
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
		log.Fatal(err)
	}

	// Perform multiple renders
	iterations := 3
	fmt.Printf("Rendering %d times...\n\n", iterations)

	for i := range iterations {
		objects, err := e.Render(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Render %d: Rendered %d objects\n", i+1, len(objects))
	}

	// Print metrics summary
	fmt.Printf("\n=== Metrics Summary ===\n\n")

	// Render-level metrics
	renderSummary := renderMetric.Summary()
	fmt.Println("Render Metrics:")
	fmt.Printf("  Total Renders: %d\n", renderSummary.TotalRenders)
	fmt.Printf("  Average Duration: %v\n", renderSummary.AverageDuration)
	fmt.Printf("  Total Objects: %d\n", renderSummary.TotalObjects)

	// Renderer-specific metrics
	rendererSummary := rendererMetric.Summary()
	fmt.Println("\nRenderer Metrics:")
	for name, stats := range rendererSummary {
		fmt.Printf("  %s:\n", name)
		fmt.Printf("    Executions: %d\n", stats.Executions)
		fmt.Printf("    Avg Duration: %v\n", stats.AverageDuration)
		fmt.Printf("    Total Objects: %d\n", stats.TotalObjects)
		fmt.Printf("    Errors: %d\n", stats.Errors)
	}

	fmt.Println("\n=== Notes ===")
	fmt.Println("- Metrics are optional and have zero overhead when not used")
	fmt.Println("- Each metric type has its own interface (client-go pattern)")
	fmt.Println("- Easy to integrate with Prometheus, OpenTelemetry, etc.")
}
