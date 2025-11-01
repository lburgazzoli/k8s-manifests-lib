package noop_test

import (
	"testing"
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/metrics/noop"

	. "github.com/onsi/gomega"
)

func TestRenderMetric(t *testing.T) {
	ctx := t.Context()

	t.Run("should not panic", func(t *testing.T) {
		g := NewWithT(t)
		m := noop.RenderMetric{}
		g.Expect(func() {
			m.Observe(ctx, 100*time.Millisecond, 10)
		}).ToNot(Panic())
	})
}

func TestRendererMetric(t *testing.T) {
	ctx := t.Context()

	t.Run("should not panic", func(t *testing.T) {
		g := NewWithT(t)
		m := noop.RendererMetric{}
		g.Expect(func() {
			m.Observe(ctx, "helm", 100*time.Millisecond, 10, nil)
		}).ToNot(Panic())
	})
}
