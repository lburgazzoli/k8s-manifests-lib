package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
)

func main() {
	fmt.Println("=== Name Transformation Example ===")
	fmt.Println("Demonstrates: Modifying object names with prefix, suffix, and replace")
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

	// Example 1: Add prefix to names
	fmt.Println("1. SetPrefix - Add 'prod-' prefix to all object names")
	prefixTransformer := name.SetPrefix("prod-")

	e1 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(prefixTransformer),
	)

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (added 'prod-' prefix)\n", len(objects1))
	if len(objects1) > 0 {
		fmt.Printf("   Example: %s\n\n", objects1[0].GetName())
	}

	// Example 2: Add suffix to names
	fmt.Println("2. SetSuffix - Add '-v2' suffix to all object names")
	suffixTransformer := name.SetSuffix("-v2")

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(suffixTransformer),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (added '-v2' suffix)\n", len(objects2))
	if len(objects2) > 0 {
		fmt.Printf("   Example: %s\n\n", objects2[0].GetName())
	}

	// Example 3: Replace substring in names
	fmt.Println("3. Replace - Replace 'staging' with 'production' in names")
	replaceTransformer := name.Replace("staging", "production")

	e3 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithTransformer(replaceTransformer),
	)

	objects3, err := e3.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Transformed %d objects (replaced 'staging' → 'production')\n", len(objects3))
	if len(objects3) > 0 {
		fmt.Printf("   Example: %s\n", objects3[0].GetName())
	}
}
