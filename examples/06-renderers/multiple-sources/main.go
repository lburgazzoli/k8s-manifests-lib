package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rs/xid"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
)

func main() {
	fmt.Println("=== Multiple Helm Sources Example ===")
	fmt.Println("Demonstrates: Rendering multiple Helm charts with a single renderer")
	fmt.Println()

	// Create a Helm renderer with multiple source charts
	// Each chart is processed independently and results are aggregated
	helmRenderer, err := helm.New([]helm.Source{
		{
			Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
			ReleaseName: "app-one",
			Values: helm.Values(map[string]any{
				"shared": map[string]any{
					"appId": xid.New().String(),
				},
			}),
		},
		{
			Repo:        "https://dapr.github.io/helm-charts",
			Chart:       "dapr",
			ReleaseName: "app-two",
			Values: helm.Values(map[string]any{
				"shared": map[string]any{
					"appId": xid.New().String(),
				},
			}),
		},
	})
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	e := engine.New(engine.WithRenderer(helmRenderer))

	objects, err := e.Render(context.Background())
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	fmt.Printf("Successfully rendered %d objects from %d Helm charts\n\n", len(objects), 2)

	// Count objects per release
	releaseCounts := make(map[string]int)
	for _, obj := range objects {
		labels := obj.GetLabels()
		if release, ok := labels["app.kubernetes.io/instance"]; ok {
			releaseCounts[release]++
		}
	}

	fmt.Println("Objects per release:")
	for release, count := range releaseCounts {
		fmt.Printf("  - %s: %d objects\n", release, count)
	}
}
