package metrics_test

import (
	"sync"
	"testing"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/memory"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/noop"

	. "github.com/onsi/gomega"
)

func TestMetricsContext(t *testing.T) {
	ctx := t.Context()

	t.Run("should store and retrieve metrics from context", func(t *testing.T) {
		g := NewWithT(t)
		m := &metrics.Metrics{
			RenderMetric:   &memory.RenderMetric{},
			RendererMetric: memory.NewRendererMetric(),
		}

		ctxWithMetrics := metrics.WithMetrics(ctx, m)
		retrieved := metrics.FromContext(ctxWithMetrics)

		g.Expect(retrieved).ToNot(BeNil())
		g.Expect(retrieved.RenderMetric).ToNot(BeNil())
		g.Expect(retrieved.RendererMetric).ToNot(BeNil())
	})

	t.Run("should return nil when metrics not in context", func(t *testing.T) {
		g := NewWithT(t)
		retrieved := metrics.FromContext(ctx)
		g.Expect(retrieved).To(BeNil())
	})

	t.Run("should allow nil metrics fields", func(t *testing.T) {
		g := NewWithT(t)
		m := &metrics.Metrics{
			RenderMetric: &memory.RenderMetric{},
		}

		ctxWithMetrics := metrics.WithMetrics(ctx, m)
		retrieved := metrics.FromContext(ctxWithMetrics)

		g.Expect(retrieved).ToNot(BeNil())
		g.Expect(retrieved.RenderMetric).ToNot(BeNil())
		g.Expect(retrieved.RendererMetric).To(BeNil())
	})

	t.Run("should work with noop metrics", func(t *testing.T) {
		g := NewWithT(t)
		m := &metrics.Metrics{
			RenderMetric:   noop.RenderMetric{},
			RendererMetric: noop.RendererMetric{},
		}

		ctxWithMetrics := metrics.WithMetrics(ctx, m)
		retrieved := metrics.FromContext(ctxWithMetrics)

		g.Expect(retrieved).ToNot(BeNil())
		g.Expect(retrieved.RenderMetric).ToNot(BeNil())
		g.Expect(retrieved.RendererMetric).ToNot(BeNil())
	})
}

func TestObserveRendererNilSafety(t *testing.T) {
	t.Run("should safely no-op when no metrics in context", func(t *testing.T) {
		ctx := t.Context()

		metrics.ObserveRenderer(ctx, "helm", 100*time.Millisecond, 10, nil)
	})

	t.Run("should safely no-op when RendererMetric is nil", func(t *testing.T) {
		m := &metrics.Metrics{
			RenderMetric:   &memory.RenderMetric{},
			RendererMetric: nil,
		}
		ctx := metrics.WithMetrics(t.Context(), m)

		metrics.ObserveRenderer(ctx, "helm", 100*time.Millisecond, 10, nil)
	})
}

func TestObserveRenderNilSafety(t *testing.T) {
	t.Run("should safely no-op when RenderMetric is nil", func(t *testing.T) {
		m := &metrics.Metrics{
			RenderMetric:   nil,
			RendererMetric: memory.NewRendererMetric(),
		}
		ctx := metrics.WithMetrics(t.Context(), m)

		metrics.ObserveRender(ctx, 100*time.Millisecond, 10)
	})
}

func TestThreadSafety(t *testing.T) {

	t.Run("should be thread-safe for concurrent renderer observations", func(t *testing.T) {
		g := NewWithT(t)
		m := &metrics.Metrics{
			RendererMetric: memory.NewRendererMetric(),
		}
		ctx := metrics.WithMetrics(t.Context(), m)

		var wg sync.WaitGroup
		for i := range 100 {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				rendererType := []string{"helm", "kustomize", "yaml"}[id%3]
				metrics.ObserveRenderer(ctx, rendererType, time.Millisecond, 1, nil)
			}(i)
		}
		wg.Wait()

		summary := m.RendererMetric.(*memory.RendererMetric).Summary()
		totalExecs := 0
		for _, stats := range summary {
			totalExecs += stats.Executions
		}
		g.Expect(totalExecs).To(Equal(100))
	})

	t.Run("should be thread-safe for concurrent render observations", func(t *testing.T) {
		g := NewWithT(t)
		m := &metrics.Metrics{
			RenderMetric: &memory.RenderMetric{},
		}
		ctx := metrics.WithMetrics(t.Context(), m)

		var wg sync.WaitGroup
		for range 100 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				metrics.ObserveRender(ctx, time.Millisecond, 1)
			}()
		}
		wg.Wait()

		summary := m.RenderMetric.(*memory.RenderMetric).Summary()
		g.Expect(summary.TotalRenders).To(Equal(100))
		g.Expect(summary.TotalObjects).To(Equal(100))
	})
}
