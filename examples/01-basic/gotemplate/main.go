package main

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/gotemplate"
)

//go:embed templates/*.yaml.tmpl
var templatesFS embed.FS

func main() {
	fmt.Println("=== Basic Go Template Example ===")
	fmt.Println("Demonstrates: Simple Go template rendering using engine.GoTemplate() convenience function")
	fmt.Println()

	// Create an Engine with a single Go template renderer
	// Using embedded filesystem for portability
	e, err := engine.GoTemplate(gotemplate.Source{
		FS:   templatesFS,
		Path: "templates/*.yaml.tmpl", // Glob pattern to match template files
		Values: gotemplate.Values(map[string]any{
			"appName":   "my-app",
			"namespace": "default",
			"replicas":  3,
			"image": map[string]any{
				"repository": "nginx",
				"tag":        "latest",
			},
		}),
	})
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Render the templates
	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	// Print summary
	fmt.Printf("Successfully rendered %d Kubernetes objects from Go templates\n\n", len(objects))

	// Show what was rendered
	fmt.Println("Rendered objects:")
	for i, obj := range objects {
		fmt.Printf("%d. %s/%s", i+1, obj.GetKind(), obj.GetName())
		if obj.GetNamespace() != "" {
			fmt.Printf(" (namespace: %s)", obj.GetNamespace())
		}
		fmt.Println()
	}
}
