package memory

import (
	"context"
	"sync"
	"time"
)

// RenderMetric collects render metrics in memory.
type RenderMetric struct {
	mu sync.RWMutex

	TotalRenders  int
	TotalDuration time.Duration
	TotalObjects  int
}

// Observe records a render operation's metrics.
func (m *RenderMetric) Observe(_ context.Context, duration time.Duration, objectCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRenders++
	m.TotalDuration += duration
	m.TotalObjects += objectCount
}

// Summary returns a snapshot of current render metrics.
func (m *RenderMetric) Summary() RenderSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgDuration := time.Duration(0)
	if m.TotalRenders > 0 {
		avgDuration = m.TotalDuration / time.Duration(m.TotalRenders)
	}

	return RenderSummary{
		TotalRenders:    m.TotalRenders,
		AverageDuration: avgDuration,
		TotalObjects:    m.TotalObjects,
	}
}

// RenderSummary provides a snapshot of render metrics.
type RenderSummary struct {
	TotalRenders    int
	AverageDuration time.Duration
	TotalObjects    int
}

// RendererMetric collects renderer-specific metrics in memory.
type RendererMetric struct {
	mu        sync.RWMutex
	Renderers map[string]*RendererStats
}

// RendererStats holds statistics for a specific renderer type.
type RendererStats struct {
	Executions int
	Duration   time.Duration
	Objects    int
	Errors     int
}

// NewRendererMetric creates a new renderer metrics collector.
func NewRendererMetric() *RendererMetric {
	return &RendererMetric{
		Renderers: make(map[string]*RendererStats),
	}
}

// Observe records a renderer execution's metrics.
func (m *RendererMetric) Observe(
	_ context.Context,
	rendererType string,
	duration time.Duration,
	objectCount int,
	err error,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.Renderers[rendererType]; !exists {
		m.Renderers[rendererType] = &RendererStats{}
	}

	stats := m.Renderers[rendererType]
	stats.Executions++
	stats.Duration += duration
	stats.Objects += objectCount
	if err != nil {
		stats.Errors++
	}
}

// Summary returns a snapshot of current renderer metrics.
func (m *RendererMetric) Summary() map[string]RendererSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]RendererSummary)
	for name, stats := range m.Renderers {
		avgDuration := time.Duration(0)
		if stats.Executions > 0 {
			avgDuration = stats.Duration / time.Duration(stats.Executions)
		}

		result[name] = RendererSummary{
			Executions:      stats.Executions,
			AverageDuration: avgDuration,
			TotalObjects:    stats.Objects,
			Errors:          stats.Errors,
		}
	}

	return result
}

// RendererSummary provides a snapshot of metrics for a specific renderer.
type RendererSummary struct {
	Executions      int
	AverageDuration time.Duration
	TotalObjects    int
	Errors          int
}
