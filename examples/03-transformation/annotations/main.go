package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
)

func main() {
	fmt.Println("=== Annotation Transformation Example ===")
	fmt.Println("Demonstrates: Adding, updating, and removing annotations")
	fmt.Println()

	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/bitnamicharts/nginx",
			ReleaseName: "my-nginx",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Example 1: Add/Update annotations
	fmt.Println("1. Set - Add or update annotations")
	setTransformer := annotations.Set(map[string]string{
		"description":          "NGINX web server",
		"contact":              "team@example.com",
		"deployed-by":          "k8s-manifests-lib",
		"prometheus.io/scrape": "true",
		"prometheus.io/port":   "9113",
	})

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(setTransformer),
	)

	if err != nil {
		log.Fatal(err)
	}

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (added 5 annotations)\n", len(objects1))
	if len(objects1) > 0 {
		fmt.Printf("   Example annotations: %v\n\n", objects1[0].GetAnnotations())
	}

	// Example 2: Remove specific annotations
	fmt.Println("2. Remove - Remove specific annotation keys")
	removeTransformer := annotations.Remove("temp-note", "debug-info")

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(removeTransformer),
	)

	if err != nil {
		log.Fatal(err)
	}

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (removed temp annotations)\n\n", len(objects2))

	// Example 3: Remove annotations conditionally
	fmt.Println("3. RemoveIf - Remove annotations matching a condition")
	removeIfTransformer := annotations.RemoveIf(func(key string, value string) bool {
		// Remove annotations with "delete-me" or "temporary" values
		return value == "delete-me" || value == "temporary"
	})

	e3, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(removeIfTransformer),
	)

	if err != nil {
		log.Fatal(err)
	}

	objects3, err := e3.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (removed temp-value annotations)\n", len(objects3))
}
