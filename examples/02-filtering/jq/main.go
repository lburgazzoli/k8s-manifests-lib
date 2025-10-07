package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/jq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	jqutil "github.com/lburgazzoli/k8s-manifests-lib/pkg/util/jq"
)

func main() {
	fmt.Println("=== JQ Filtering Example ===")
	fmt.Println("Demonstrates: Filtering objects using JQ expressions")
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

	// Example 1: Filter by API version
	fmt.Println("1. API Version - Keep only apps/v1 resources")
	appsV1Filter, err := jq.Filter(`.apiVersion == "apps/v1"`)
	if err != nil {
		log.Fatal(err)
	}

	e1 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(appsV1Filter),
	)

	objects1, err := e1.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d apps/v1 objects\n\n", len(objects1))

	// Example 2: Complex boolean expression
	fmt.Println("2. Boolean Logic - Keep Deployments OR Services")
	orFilter, err := jq.Filter(`.kind == "Deployment" or .kind == "Service"`)
	if err != nil {
		log.Fatal(err)
	}

	e2 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(orFilter),
	)

	objects2, err := e2.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d objects (Deployments or Services)\n\n", len(objects2))

	// Example 3: JQ with variables
	fmt.Println("3. With Variables - Filter by dynamic kind")
	varFilter, err := jq.Filter(
		`.kind == $expectedKind`,
		jqutil.WithVariable("expectedKind", "Deployment"),
	)
	if err != nil {
		log.Fatal(err)
	}

	e3 := engine.New(
		engine.WithRenderer(helmRenderer),
		engine.WithFilter(varFilter),
	)

	objects3, err := e3.Render(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Rendered %d Deployments (using JQ variable)\n", len(objects3))
}
