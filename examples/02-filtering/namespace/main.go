package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Namespace Filtering Example ===")
	fmt.Println("Demonstrates: Filtering objects by namespace")
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

	// Example 1: Include only specific namespaces
	fmt.Println("1. Include Filter - Keep only objects in 'production' and 'staging'")
	includeFilter := namespace.Filter("production", "staging")

	e1, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(includeFilter),
	)
	if err != nil {
		log.Fatal(err)
	}

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d objects from production/staging namespaces\n\n", len(objects1))

	// Example 2: Exclude specific namespaces
	fmt.Println("2. Exclude Filter - Exclude system namespaces")
	excludeFilter := namespace.Exclude("kube-system", "kube-public", "kube-node-lease")

	e2, err := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(excludeFilter),
	)
	if err != nil {
		log.Fatal(err)
	}

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d objects (excluding system namespaces)\n", len(objects2))
}
