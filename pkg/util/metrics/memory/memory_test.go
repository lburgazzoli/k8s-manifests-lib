package memory_test

import (
	"errors"
	"testing"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/memory"

	. "github.com/onsi/gomega"
)

func TestRenderMetric(t *testing.T) {
	ctx := t.Context()

	t.Run("should record single render", func(t *testing.T) {
		g := NewWithT(t)
		m := &memory.RenderMetric{}
		m.Observe(ctx, 100*time.Millisecond, 10)

		summary := m.Summary()
		g.Expect(summary.TotalRenders).To(Equal(1))
		g.Expect(summary.TotalObjects).To(Equal(10))
		g.Expect(summary.AverageDuration).To(Equal(100 * time.Millisecond))
	})

	t.Run("should record multiple renders", func(t *testing.T) {
		g := NewWithT(t)
		m := &memory.RenderMetric{}
		m.Observe(ctx, 100*time.Millisecond, 10)
		m.Observe(ctx, 200*time.Millisecond, 15)
		m.Observe(ctx, 300*time.Millisecond, 5)

		summary := m.Summary()
		g.Expect(summary.TotalRenders).To(Equal(3))
		g.Expect(summary.TotalObjects).To(Equal(30))
		g.Expect(summary.AverageDuration).To(Equal(200 * time.Millisecond))
	})

	t.Run("should handle zero renders", func(t *testing.T) {
		g := NewWithT(t)
		m := &memory.RenderMetric{}

		summary := m.Summary()
		g.Expect(summary.TotalRenders).To(Equal(0))
		g.Expect(summary.TotalObjects).To(Equal(0))
		g.Expect(summary.AverageDuration).To(Equal(time.Duration(0)))
	})
}

func TestRendererMetric(t *testing.T) {
	ctx := t.Context()

	t.Run("should record single renderer execution", func(t *testing.T) {
		g := NewWithT(t)
		m := memory.NewRendererMetric()
		m.Observe(ctx, "helm", 100*time.Millisecond, 10, nil)

		summary := m.Summary()
		g.Expect(summary).To(HaveKey("helm"))

		helmStats := summary["helm"]
		g.Expect(helmStats.Executions).To(Equal(1))
		g.Expect(helmStats.TotalObjects).To(Equal(10))
		g.Expect(helmStats.AverageDuration).To(Equal(100 * time.Millisecond))
		g.Expect(helmStats.Errors).To(Equal(0))
	})

	t.Run("should record multiple renderer types", func(t *testing.T) {
		g := NewWithT(t)
		m := memory.NewRendererMetric()
		m.Observe(ctx, "helm", 100*time.Millisecond, 10, nil)
		m.Observe(ctx, "kustomize", 200*time.Millisecond, 15, nil)

		summary := m.Summary()
		g.Expect(summary).To(HaveKey("helm"))
		g.Expect(summary).To(HaveKey("kustomize"))

		g.Expect(summary["helm"].Executions).To(Equal(1))
		g.Expect(summary["kustomize"].Executions).To(Equal(1))
	})

	t.Run("should aggregate multiple executions of same renderer", func(t *testing.T) {
		g := NewWithT(t)
		m := memory.NewRendererMetric()
		m.Observe(ctx, "helm", 100*time.Millisecond, 10, nil)
		m.Observe(ctx, "helm", 200*time.Millisecond, 20, nil)
		m.Observe(ctx, "helm", 300*time.Millisecond, 30, nil)

		summary := m.Summary()
		helmStats := summary["helm"]

		g.Expect(helmStats.Executions).To(Equal(3))
		g.Expect(helmStats.TotalObjects).To(Equal(60))
		g.Expect(helmStats.AverageDuration).To(Equal(200 * time.Millisecond))
		g.Expect(helmStats.Errors).To(Equal(0))
	})

	t.Run("should track errors", func(t *testing.T) {
		g := NewWithT(t)
		m := memory.NewRendererMetric()
		m.Observe(ctx, "helm", 100*time.Millisecond, 0, errors.New("test error"))
		m.Observe(ctx, "helm", 200*time.Millisecond, 10, nil)
		m.Observe(ctx, "helm", 150*time.Millisecond, 0, errors.New("another error"))

		summary := m.Summary()
		helmStats := summary["helm"]

		g.Expect(helmStats.Executions).To(Equal(3))
		g.Expect(helmStats.Errors).To(Equal(2))
		g.Expect(helmStats.TotalObjects).To(Equal(10))
	})
}
