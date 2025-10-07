package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
)

func main() {
	fmt.Println("=== Namespace Transformation Example ===\n")
	fmt.Println("Demonstrates: Setting and ensuring namespaces on objects")
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

	// Example 1: Force namespace unconditionally
	fmt.Println("1. Set - Force namespace to 'production'")
	setTransformer := namespace.Set("production")

	e1 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(setTransformer),
	)

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (namespace → production)\n", len(objects1))
	if len(objects1) > 0 {
		fmt.Printf("   Example: %s/%s now in '%s' namespace\n\n",
			objects1[0].GetKind(), objects1[0].GetName(), objects1[0].GetNamespace())
	}

	// Example 2: Set default namespace only if empty
	fmt.Println("2. EnsureDefault - Set namespace only if not already set")
	ensureTransformer := namespace.EnsureDefault("default")

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(ensureTransformer),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (namespace → default if empty)\n", len(objects2))
	fmt.Println("   Objects with existing namespaces are not modified")
	fmt.Println("   Objects without namespace get 'default'")
}
