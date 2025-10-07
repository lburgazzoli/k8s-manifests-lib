# Kubernetes Manifests Library

## Introduction

The Kubernetes Manifests Library is a Go-based toolkit designed to simplify the management, transformation, and rendering of Kubernetes manifests. It provides a robust set of utilities for working with Kubernetes resources programmatically, making it easier to generate, modify, and validate Kubernetes configurations.

## Use Cases

**This library is designed to be embedded in Go applications and libraries**, not as a replacement for kubectl, Helm, or Kustomize CLIs.

### When to Use This Library

Use k8s-manifests-lib when you need to **programmatically** work with Kubernetes manifests in your Go code:

- **Kubernetes Operators/Controllers**: Deploy and manage components using Helm charts, Kustomize, or YAML templates
- **GitOps Tools**: Process, transform, and validate manifests before applying them
- **CI/CD Pipelines**: Customize manifests based on environment, inject labels/annotations, filter resources
- **Multi-tenant Platforms**: Generate tenant-specific configurations from shared templates
- **Custom Deployment Tools**: Build application-specific deployment logic with manifest rendering

### When NOT to Use This Library

- **Ad-hoc manifest operations**: Use `kubectl`, `helm`, or `kustomize` CLI directly
- **Manual deployments**: Standard CLI tools are more appropriate
- **Simple scripting**: Shell scripts with CLI tools may be simpler

### Key Differentiator

Unlike CLI tools, this library provides a **Go API** for manifest operations, enabling:
- Type-safe configuration
- Programmatic filtering and transformation
- Integration into larger Go applications
- Testable manifest rendering logic
- Complex composition of multiple sources

## Features

* Manifest rendering from multiple sources (Helm, Kustomize, Go templates, YAML)
* Resource transformation and filtering with JQ expressions
* Filter composition with boolean logic (Or, And, Not) and conditionals
* Transformer composition with chaining, conditionals, and multi-branch logic
* Built-in metadata filters (namespace, labels, annotations, name)
* Built-in metadata transformers (namespace, labels, annotations, name)
* Type-safe Kubernetes resource definitions
* Three-level filtering/transformation pipeline (renderer-specific, engine-level, render-time)
* Extensible engine for custom processing
* Built-in caching with TTL support and automatic deep cloning
* Parallel rendering for I/O-bound renderers
* Functional options pattern for flexible configuration

## Installation

```bash
go get github.com/lburgazzoli/k8s-manifests-lib
```

## Quick Start

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
            Chart:       "oci://registry.example.com/my-chart:1.0.0",
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
        engine.WithTransformer(labels.Set(map[string]string{
            "app.kubernetes.io/managed-by": "my-operator",
        })),
    )

    // Render with additional render-time options
    ctx := context.Background()
    objects, err := e.Render(ctx,
        // Add a render-time transformer to add an environment label
        engine.WithRenderTransformer(labels.Set(map[string]string{
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

This example demonstrates:
* Rendering Helm charts from OCI registries
* Filtering objects with JQ expressions
* Transforming objects with label additions
* Three-level filtering/transformation pipeline (renderer → engine → render-time)

## Examples

For specific use cases and patterns, see the [examples directory](examples/):

- **[01-basic/](examples/01-basic/)** - Simple single-renderer examples ([helm](examples/01-basic/helm/main.go), [kustomize](examples/01-basic/kustomize/main.go), [yaml](examples/01-basic/yaml/main.go), [gotemplate](examples/01-basic/gotemplate/main.go))
- **[02-filtering/](examples/02-filtering/)** - Filtering by [namespace](examples/02-filtering/namespace/main.go), [labels](examples/02-filtering/labels/main.go), [GVK](examples/02-filtering/gvk/main.go), [JQ](examples/02-filtering/jq/main.go)
- **[03-transformation/](examples/03-transformation/)** - Transforming [labels](examples/03-transformation/labels/main.go), [annotations](examples/03-transformation/annotations/main.go), [namespace](examples/03-transformation/namespace/main.go), [name](examples/03-transformation/name/main.go)
- **[04-composition/](examples/04-composition/)** - [Boolean logic](examples/04-composition/filter-boolean/main.go), [conditionals](examples/04-composition/filter-conditional/main.go), [chaining](examples/04-composition/transformer-chain/main.go), [switch](examples/04-composition/transformer-switch/main.go)
- **[05-advanced/](examples/05-advanced/)** - [Three-level pipeline](examples/05-advanced/three-level-pipeline/main.go), [multi-environment](examples/05-advanced/multi-environment/main.go), [conditional transformations](examples/05-advanced/conditional-transformations/main.go), [complex nested](examples/05-advanced/complex-nested/main.go)
- **[06-renderers/](examples/06-renderers/)** - [Multiple sources](examples/06-renderers/multiple-sources/main.go), [multiple renderers](examples/06-renderers/multiple-renderers/main.go), [dynamic values](examples/06-renderers/dynamic-values/main.go)
- **[07-caching/](examples/07-caching/)** - [Basic cache](examples/07-caching/basic/main.go), [performance benchmarks](examples/07-caching/performance/main.go)
- **[08-parallel/](examples/08-parallel/)** - [Parallel rendering](examples/08-parallel/main.go) for improved performance with I/O-bound renderers

See the [Examples README](examples/README.md) for a complete catalog with a recommended learning path.

Each example is runnable: `go run examples/<category>/<name>/main.go`

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
