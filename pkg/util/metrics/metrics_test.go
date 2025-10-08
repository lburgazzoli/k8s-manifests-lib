package metrics_test

import (
	"context"
	"testing"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/memory"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/noop"

	. "github.com/onsi/gomega"
)

func TestMetricsContext(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	t.Run("should store and retrieve metrics from context", func(t *testing.T) {
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
		retrieved := metrics.FromContext(ctx)
		g.Expect(retrieved).To(BeNil())
	})

	t.Run("should allow nil metrics fields", func(t *testing.T) {
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
