package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
)

func main() {
	fmt.Println("=== Label Transformation Example ===\n")
	fmt.Println("Demonstrates: Adding, updating, and removing labels")
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

	// Example 1: Add/Update labels
	fmt.Println("1. Set - Add or update labels")
	setTransformer := labels.Set(map[string]string{
		"env":                              "production",
		"tier":                             "frontend",
		"app.kubernetes.io/managed-by":     "k8s-manifests-lib",
		"app.kubernetes.io/part-of":        "nginx-stack",
	})

	e1 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(setTransformer),
	)

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (added 4 labels)\n", len(objects1))
	if len(objects1) > 0 {
		fmt.Printf("   Example labels: %v\n\n", objects1[0].GetLabels())
	}

	// Example 2: Remove specific labels
	fmt.Println("2. Remove - Remove specific label keys")
	removeTransformer := labels.Remove("temp", "debug", "test-only")

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(removeTransformer),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (removed temp/debug/test-only labels)\n\n", len(objects2))

	// Example 3: Remove labels conditionally
	fmt.Println("3. RemoveIf - Remove labels matching a condition")
	removeIfTransformer := labels.RemoveIf(func(key string, value string) bool {
		// Remove all labels with 'temp-' prefix
		return strings.HasPrefix(key, "temp-")
	})

	e3 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(removeIfTransformer),
	)

	objects3, err := e3.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (removed labels with 'temp-' prefix)\n", len(objects3))
}
