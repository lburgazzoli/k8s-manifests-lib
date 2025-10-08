package noop

import (
	"context"
	"time"
)

type RenderMetric struct{}

func (RenderMetric) Observe(_ context.Context, _ time.Duration, _ int) {
}

type RendererMetric struct{}

func (RendererMetric) Observe(_ context.Context, _ string, _ time.Duration, _ int, _ error) {
}
