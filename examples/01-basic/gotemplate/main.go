package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/examples/internal/logger"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
)

//go:embed templates/*.yaml.tmpl
var templatesFS embed.FS

func main() {
	ctx := logger.WithLogger(context.Background(), &logger.StdoutLogger{})
	if err := Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func Run(ctx context.Context) error {
	l := logger.FromContext(ctx)
	l.Log("=== Basic Go Template Example ===")
	l.Log("Demonstrates: Simple Go template rendering using engine.GoTemplate() convenience function")
	l.Log("")

	// Create an Engine with a single Go template renderer
	// Using embedded filesystem for portability
	e, err := engine.GoTemplate(gotemplate.Source{
		FS:   templatesFS,
		Path: "templates/*.yaml.tmpl", // Glob pattern to match template files
		Values: gotemplate.Values(map[string]any{
			"Name":      "my-app",
			"Namespace": "default",
			"Replicas":  3,
			"Image":     "nginx:latest",
			"Port":      80,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Render the templates
	objects, err := e.Render(ctx)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Print summary
	l.Logf("Successfully rendered %d Kubernetes objects from Go templates\n\n", len(objects))

	// Show what was rendered
	l.Log("Rendered objects:")
	for i, obj := range objects {
		l.Logf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			l.Logf(" (namespace: %s)", obj.GetNamespace())
		}
		l.Log("")
	}

	return nil
}
