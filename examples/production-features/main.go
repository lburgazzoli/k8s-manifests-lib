package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/memory"
)

//go:embed manifests/*.yaml
var manifestsFS embed.FS

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Production Features ===")
	l.Log("Demonstrates: Caching, Parallel Rendering, Metrics, and Source Annotations")
	l.Log("")

	// Feature 1: Caching - Improve performance with TTL-based caching
	l.Log("1. Caching: Enable renderer-level caching with 5-minute TTL")
	helmRenderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
				ReleaseName: "my-nginx",
				Values: helm.Values(map[string]any{
					"replicaCount": 3,
				}),
			},
		},
		helm.WithCache(cache.WithTTL(5*time.Minute)), // Enable caching
		helm.WithSourceAnnotations(true),             // Feature 4: Source tracking
	)
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Add a simple YAML renderer with source annotations
	yamlRenderer, err := yaml.New(
		[]yaml.Source{
			{
				FS:   manifestsFS,
				Path: "manifests/*.yaml",
			},
		},
		yaml.WithSourceAnnotations(true), // Feature 4: Source tracking
	)
	if err != nil {
		return fmt.Errorf("failed to create yaml renderer: %w", err)
	}

	// Feature 3: Metrics - Collect rendering performance metrics
	l.Log("2. Metrics: Enable performance monitoring")
	renderMetric := &memory.RenderMetric{}
	rendererMetric := memory.NewRendererMetric()
	ctx = metrics.WithMetrics(ctx, &metrics.Metrics{
		RenderMetric:   renderMetric,
		RendererMetric: rendererMetric,
	})

	// Feature 2: Parallel Rendering - Process multiple renderers concurrently
	l.Log("3. Parallel Rendering: Process multiple sources concurrently")
	l.Log("4. Source Annotations: Track which file/chart produced each object")
	l.Log("")

	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithRenderer(yamlRenderer),
		engine.WithParallel(true), // Feature 2: Enable parallel rendering
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// First render (cache miss for Helm)
	l.Log("=== First Render ===")
	start := time.Now()
	objects1, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	duration1 := time.Since(start)
	l.Logf("Rendered %d objects in %v (cache miss)\n\n", len(objects1), duration1)

	// Second render (cache hit for Helm)
	l.Log("=== Second Render ===")
	start = time.Now()
	objects2, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	duration2 := time.Since(start)
	l.Logf("Rendered %d objects in %v (cache hit - ~%0.1fx faster)\n\n",
		len(objects2), duration2, float64(duration1)/float64(duration2))

	// Show source annotations
	l.Log("=== Source Annotations ===")
	l.Log("Each object tracks its origin:")
	for i, obj := range objects1 {
		if i >= 3 {
			l.Logf("... and %d more objects\n", len(objects1)-3)

			break
		}
		annotations := obj.GetAnnotations()
		l.Logf("%d. %s/%s\n", i+1, obj.GetKind(), obj.GetName())
		if sourceType, ok := annotations[types.AnnotationSourceType]; ok {
			l.Logf("   Source Type: %s\n", sourceType)
		}
		if sourcePath, ok := annotations[types.AnnotationSourcePath]; ok {
			l.Logf("   Source Path: %s\n", sourcePath)
		}
		if sourceFile, ok := annotations[types.AnnotationSourceFile]; ok {
			l.Logf("   Source File: %s\n", sourceFile)
		}
		l.Log("")
	}

	// Show metrics
	l.Log("=== Performance Metrics ===")
	renderSummary := renderMetric.Summary()
	l.Logf("Total Renders: %d\n", renderSummary.TotalRenders)
	l.Logf("Average Duration: %v\n", renderSummary.AverageDuration)
	l.Logf("Total Objects: %d\n\n", renderSummary.TotalObjects)

	l.Log("Renderer Metrics:")
	for name, summary := range rendererMetric.Summary() {
		l.Logf("  %s:\n", name)
		l.Logf("    Executions: %d\n", summary.Executions)
		l.Logf("    Avg Duration: %v\n", summary.AverageDuration)
		l.Logf("    Total Objects: %d\n", summary.TotalObjects)
		l.Logf("    Errors: %d\n", summary.Errors)
	}

	l.Log("")
	l.Log("=== Summary ===")
	l.Log("✓ Caching: Helm results cached for 5 minutes")
	l.Log("✓ Parallel: Helm and YAML renderers run concurrently")
	l.Log("✓ Metrics: Performance data collected for observability")
	l.Log("✓ Source Tracking: Each object annotated with its origin")

	return nil
}
