package main_test

import (
	"context"
	"testing"
	"time"

	example "github.com/lburgazzoli/k8s-manifests-lib/examples/07-caching/basic"
	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ctx = logger.WithLogger(ctx, t)

	if err := example.Run(ctx); err != nil {
		t.Fatalf("Run() failed: %v", err)
	}
}
