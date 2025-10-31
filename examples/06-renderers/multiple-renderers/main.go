package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
)

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Multiple Renderers Example ===")
	l.Log("Demonstrates: Combining Helm, Kustomize, and YAML renderers")
	l.Log()

	// Create a Helm renderer
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create helm renderer: %w", err)
	}

	// Create a Kustomize renderer
	kustomizeRenderer, err := kustomize.New([]kustomize.Source{
		{
			Path: "./kustomization-example",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create kustomize renderer: %w", err)
	}

	// Create a YAML renderer
	yamlRenderer, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("./manifests"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create yaml renderer: %w", err)
	}

	// Combine all three renderers in a single engine
	e, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithRenderer(kustomizeRenderer),
		engine.WithRenderer(yamlRenderer),
	)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	l.Logf("Successfully rendered %d total objects from all renderers\n\n", len(objects))

	// Count objects by kind
	kindCounts := make(map[string]int)
	for _, obj := range objects {
		kindCounts[obj.GetKind()]++
	}

	l.Log("Objects by kind:")
	for kind, count := range kindCounts {
		l.Logf("  - %d %s(s)\n", count, kind)
	}

	return nil
}
