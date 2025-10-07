package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rs/xid"
	"gopkg.in/yaml.v3"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/jq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

func main() {
	// Create a Helm renderer for the Dapr chart from OCI registry
	helmRenderer, err := helm.New(
		[]helm.Source{
			{
				Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
				ReleaseName: "foo",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": xid.New().String(),
					},
				}),
			},
			{
				Repo:        "https://dapr.github.io/helm-charts",
				Chart:       "dapr",
				ReleaseName: "bar",
				Values: helm.Values(map[string]any{
					"shared": map[string]any{
						"appId": xid.New().String(),
					},
				}),
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to create Helm renderer: %v", err)
	}

	// Create a JQ filter to only keep objects from apps/v1 API group
	// This will be applied at engine-level (after renderer processing)
	appsV1Filter, err := jq.Filter(`.apiVersion == "apps/v1"`)
	if err != nil {
		log.Fatalf("Failed to create apps/v1 filter: %v", err)
	}

	// Create the engine with the apps/v1 filter applied at engine-level
	// The deployment filter was already applied at renderer-level
	e := engine.New(
		engine.WithRenderer(helmRenderer),
		// Engine-level filter: Only keep objects from apps/v1 API group
		engine.WithFilter(appsV1Filter),
		// Add a transformer to add a common label
		engine.WithTransformer(labels.Set(map[string]string{
			"app.kubernetes.io/managed-by": "k8s-manifests-lib",
		})),
	)

	// Create a context
	ctx := context.Background()

	// Create a JQ filter to only keep DaemonSet or StatefulSet objects
	appFilter, err := jq.Filter(`.kind == "DaemonSet" or .kind == "StatefulSet"`)
	if err != nil {
		log.Fatalf("Failed to create Deployment filter: %v", err)
	}

	// Render with additional render-time options (using struct-based options)
	objects, err := e.Render(
		ctx,
		engine.RenderOptions{
			// Add a render-time filter to keep only DaemonSets
			Filters: []types.Filter{
				appFilter,
			},
			// Add a render-time transformer to add an environment label
			Transformers: []types.Transformer{
				labels.Set(map[string]string{
					"environment": "production",
					"chart":       "dapr",
				}),
			},
		},
	)
	if err != nil {
		log.Fatalf("Failed to render: %v", err)
	}

	// Print the results
	fmt.Printf("\nFiltered Results (only DaemonSet or StatefulSet):\n")
	fmt.Printf("Found %d object(s):\n\n", len(objects))

	for _, obj := range objects {
		out, err := yaml.Marshal(obj.Object)
		if err != nil {
			log.Fatalf("Failed to marshal: %v", err)
		}

		fmt.Println(string(out))
	}
}
