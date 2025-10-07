# Kubernetes Manifests Library

## Introduction

The Kubernetes Manifests Library is a Go-based toolkit designed to simplify the management, transformation, and rendering of Kubernetes manifests. It provides a robust set of utilities for working with Kubernetes resources programmatically, making it easier to generate, modify, and validate Kubernetes configurations.

## Features

* Manifest rendering from multiple sources (Helm, Kustomize, Go templates, YAML)
* Resource transformation and filtering with JQ expressions
* Type-safe Kubernetes resource definitions
* Three-level filtering/transformation pipeline (renderer-specific, engine-level, render-time)
* Extensible engine for custom processing
* Built-in caching with TTL support and automatic deep cloning
* Functional options pattern for flexible configuration

## Installation

```bash
go get github.com/lburgazzoli/k8s-manifests-lib
```

## Quick Start

Here's a comprehensive example that demonstrates how to use the library to render Kubernetes manifests from a Helm chart, with filtering and transformation:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/jq"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
)

func main() {
    // Create a Helm renderer for a chart
    helmRenderer, err := helm.New([]helm.Source{
        {
            Chart:       "oci://registry.example.com/my-chart:1.0.0", // or "/path/to/chart"
            ReleaseName: "my-release",
            Values: helm.Values(map[string]any{
                "replicaCount": 3,
                "image": map[string]any{
                    "repository": "nginx",
                    "tag":        "latest",
                },
            }),
        },
    })
    if err != nil {
        log.Fatalf("Failed to create Helm renderer: %v", err)
    }

    // Create a JQ filter for resource selection
    deploymentFilter, err := jq.Filter(`.kind == "Deployment"`)
    if err != nil {
        log.Fatalf("Failed to create deployment filter: %v", err)
    }

    // Create the engine with initial configuration
    e := engine.New(
        // Add the Helm renderer
        engine.WithRenderer(helmRenderer),
        // Add a filter to only keep Deployments
        engine.WithFilter(deploymentFilter),
        // Add a transformer to add a common label
        engine.WithTransformer(labels.Transform(map[string]string{
            "app.kubernetes.io/managed-by": "my-operator",
        })),
    )

    // Render with additional render-time options
    ctx := context.Background()
    objects, err := e.Render(ctx,
        // Add a render-time transformer to add an environment label
        engine.WithRenderTransformer(labels.Transform(map[string]string{
            "environment": "production",
        })),
    )
    if err != nil {
        log.Fatalf("Failed to render: %v", err)
    }

    // Print the results
    fmt.Printf("Rendered %d objects:\n", len(objects))
    for _, obj := range objects {
        fmt.Printf("- %s/%s (%s)\n", obj.GetKind(), obj.GetName(), obj.GetNamespace())
        fmt.Printf("  Labels: %v\n", obj.GetLabels())
    }
}
```

This example demonstrates several key features of the library:

* Rendering manifests from a Helm chart with custom values
* Using JQ filters to select resources by kind
* Adding labels to resources using transformers
* Combining engine-level and render-time options
* Error handling and context support

## Using Cache for Performance

Renderers support optional caching to improve performance when rendering the same manifests multiple times:

```go
package main

import (
    "context"
    "time"

    "github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/renderer/helm"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
)

func main() {
    // Create a Helm renderer with caching enabled
    helmRenderer, _ := helm.New(
        []helm.Source{{
            Chart:       "oci://registry.example.com/my-chart:1.0.0",
            ReleaseName: "my-release",
            Values: helm.Values(map[string]any{"replicaCount": 3}),
        }},
        // Enable caching with 5-minute TTL
        helm.WithCache(cache.WithTTL(5 * time.Minute)),
    )

    e := engine.New(engine.WithRenderer(helmRenderer))

    ctx := context.Background()

    // First render: cache miss, renders from source
    objects1, _ := e.Render(ctx)

    // Second render: cache hit, returns cached results (with automatic deep clone)
    objects2, _ := e.Render(ctx)

    // Modifying objects2 won't affect the cache due to automatic deep cloning
    objects2[0].SetName("modified")

    // Third render: still gets original cached values
    objects3, _ := e.Render(ctx)
}
```

The cache automatically:
* Deep clones objects on get/set to prevent cache pollution
* Expires entries based on TTL
* Cleans up expired entries on `Sync()` calls (lazy expiration)

## Multiple Renderers

Combine multiple renderers to aggregate manifests from different sources:

```go
helmRenderer, _ := helm.New([]helm.Source{{
    Chart:       "oci://registry.example.com/helm-chart:1.0.0",
    ReleaseName: "my-release",
}})

kustomizeRenderer, _ := kustomize.New([]kustomize.Source{{
    Path: "/path/to/kustomization",
}})

yamlRenderer, _ := yaml.New([]yaml.Source{{
    FS:   os.DirFS("/path/to/manifests"),
    Path: "*.yaml",
}})

e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithRenderer(kustomizeRenderer),
    engine.WithRenderer(yamlRenderer),
)

// Render aggregates objects from all three renderers
objects, _ := e.Render(context.Background())
```

## Three-Level Filtering/Transformation

The library provides three levels of filtering and transformation:

### 1. Renderer-Specific (Applied first, inside each renderer)

```go
helmRenderer, _ := helm.New(
    []helm.Source{{...}},
    helm.WithFilter(onlyDeploymentsFilter),         // Applied by Helm only
    helm.WithTransformer(addHelmLabelsTransformer), // Applied by Helm only
)
```

### 2. Engine-Level (Applied to all renders)

```go
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(namespaceFilter),      // Applied to ALL renders
    engine.WithTransformer(addCommonLabels), // Applied to ALL renders
)
```

### 3. Render-Time (Applied per render call)

```go
objects, _ := e.Render(ctx,
    engine.WithRenderFilter(kindFilter),               // Applied only to this render
    engine.WithRenderTransformer(envLabelTransformer), // Applied only to this render
)
```

## Dynamic Values

Use dynamic value functions for runtime configuration:

```go
helmRenderer, _ := helm.New([]helm.Source{{
    Chart:       "oci://registry.example.com/my-chart:1.0.0",
    ReleaseName: "dynamic-release",
    Values: func(ctx context.Context) (map[string]any, error) {
        // Fetch values dynamically at render time
        config, err := fetchConfigFromAPI(ctx)
        if err != nil {
            return nil, err
        }
        return map[string]any{
            "appId":     generateID(),
            "timestamp": time.Now().Unix(),
            "config":    config,
        }, nil
    },
}})
```

## Project Structure

| Directory | Description |
|-----------|-------------|
| `pkg/` | Main package directory containing all library code |
| `pkg/types/` | Core type definitions (Renderer, Filter, Transformer) |
| `pkg/renderer/` | Renderer implementations (helm, kustomize, gotemplate, yaml, mem) |
| `pkg/transformer/` | Resource transformation utilities (jq, labels, annotations) |
| `pkg/filter/` | Resource filtering utilities (jq, gvk) |
| `pkg/engine/` | Core processing engine |
| `pkg/util/` | Common utility functions and cache implementation |

## Documentation

For detailed architecture and design information, see the [Design Document](docs/design.md).

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.adoc) for details.

## License

This project is licensed under the terms of the included [License file](LICENSE).

## Support

* GitHub Issues: https://github.com/lburgazzoli/k8s-manifests-lib/issues
* Documentation: https://github.com/lburgazzoli/k8s-manifests-lib/docs
