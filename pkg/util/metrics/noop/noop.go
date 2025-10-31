package noop

import (
	"context"
	"time"
)

// RenderMetric is a no-op render metrics collector that discards all observations.
type RenderMetric struct{}

// Observe does nothing; it's a no-op implementation.
func (RenderMetric) Observe(_ context.Context, _ time.Duration, _ int) {
}

// RendererMetric is a no-op renderer metrics collector that discards all observations.
type RendererMetric struct{}

// Observe does nothing; it's a no-op implementation.
func (RendererMetric) Observe(_ context.Context, _ string, _ time.Duration, _ int, _ error) {
}
