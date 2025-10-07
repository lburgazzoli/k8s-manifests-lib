package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Label Filtering Example ===\n")
	fmt.Println("Demonstrates: Filtering objects by labels")
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

	// Example 1: Check if label exists
	fmt.Println("1. HasLabel - Keep objects with 'app' label")
	hasLabelFilter := labels.HasLabel("app")

	e1 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(hasLabelFilter),
	)

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d objects with 'app' label\n\n", len(objects1))

	// Example 2: Match specific label values
	fmt.Println("2. MatchLabels - Keep objects matching exact labels")
	matchFilter := labels.MatchLabels(map[string]string{
		"app":     "nginx",
		"version": "1.0",
	})

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(matchFilter),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d objects with app=nginx AND version=1.0\n\n", len(objects2))

	// Example 3: Kubernetes label selector syntax
	fmt.Println("3. Selector - Use Kubernetes label selector syntax")
	selectorFilter, err := labels.Selector("app=nginx,tier in (frontend,backend)")
	if err != nil {
		log.Fatal(err)
	}

	e3 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(selectorFilter),
	)

	objects3, err := e3.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d objects matching selector\n", len(objects3))
}
