package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/kustomize"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/yaml"
)

func main() {
	fmt.Println("=== Multiple Renderers Example ===")
	fmt.Println("Demonstrates: Combining Helm, Kustomize, and YAML renderers")
	fmt.Println()

	// Create a Helm renderer
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	// Create a Kustomize renderer
	kustomizeRenderer, err := kustomize.New([]kustomize.Source{
		{
			Path: "./kustomization-example",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Kustomize renderer: %v", err)
	}

	// Create a YAML renderer
	yamlRenderer, err := yaml.New([]yaml.Source{
		{
			FS:   os.DirFS("./manifests"),
			Path: "*.yaml",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create YAML renderer: %v", err)
	}

	// Combine all three renderers in a single engine
	e := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithRenderer(kustomizeRenderer),
		engine.WithRenderer(yamlRenderer),
	)

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	fmt.Printf("Successfully rendered %d total objects from all renderers\n\n", len(objects))

	// Count objects by kind
	kindCounts := make(map[string]int)
	for _, obj := range objects {
		kindCounts[obj.GetKind()]++
	}

	fmt.Println("Objects by kind:")
	for kind, count := range kindCounts {
		fmt.Printf("  - %d %s(s)\n", count, kind)
	}
}
