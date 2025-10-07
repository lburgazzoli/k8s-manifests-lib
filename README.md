# Kubernetes Manifests Library

## Introduction

The Kubernetes Manifests Library is a Go-based toolkit designed to simplify the management, transformation, and rendering of Kubernetes manifests. It provides a robust set of utilities for working with Kubernetes resources programmatically, making it easier to generate, modify, and validate Kubernetes configurations.

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

## Advanced Filtering and Transformation

### Filter Composition

Build complex filter logic from simple filters using boolean combinators:

```go
import (
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/annotations"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/namespace"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/labels"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/name"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/filter/meta/gvk"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
)

// Keep production Deployments OR staging Services
complexFilter := filter.Or(
    filter.And(
        namespace.Filter("production"),
        gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
    ),
    filter.And(
        namespace.Filter("staging"),
        gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
    ),
)

// Exclude system namespaces
systemFilter := filter.Not(
    namespace.Filter("kube-system", "kube-public", "kube-node-lease"),
)

// Conditional filtering: only apply label filter if in production
productionFilter := filter.If(
    namespace.Filter("production"),
    labels.MatchLabels(map[string]string{"tier": "critical"}),
)

e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(complexFilter),
    engine.WithFilter(systemFilter),
)
```

### Built-in Metadata Filters

```go
// Namespace filters
nsFilter := namespace.Filter("production", "staging")
nsExclude := namespace.Exclude("kube-system", "default")

// Label filters
hasLabel := labels.HasLabel("environment")
hasMultiple := labels.HasLabels("app", "version", "tier")
matchLabels := labels.MatchLabels(map[string]string{
    "app": "nginx",
    "env": "prod",
})
selector, _ := labels.Selector("app=nginx,tier in (frontend,backend)")

// Name filters
exactName := name.Exact("nginx-deployment", "redis-service")
prefixName := name.Prefix("app-")
suffixName := name.Suffix("-prod")
regexName, _ := name.Regex(`^(nginx|apache)-.*$`)

// Annotation filters
hasAnnotation := annotations.HasAnnotation("owner")
matchAnnotations := annotations.MatchAnnotations(map[string]string{
    "managed-by": "k8s-manifests-lib",
})
```

### Transformer Composition

Chain transformers and apply them conditionally:

```go
import (
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer"
    nstrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/namespace"
    nametrans "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/name"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/labels"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/transformer/meta/annotations"
)

// Chain multiple transformers in sequence
chainedTransformer := transformer.Chain(
    nstrans.EnsureDefault("default"),
    labels.Set(map[string]string{
        "app.kubernetes.io/managed-by": "k8s-manifests-lib",
    }),
    annotations.Set(map[string]string{
        "rendered-at": time.Now().Format(time.RFC3339),
    }),
)

// Apply transformer conditionally
conditionalTransformer := transformer.If(
    namespace.Filter("production"),
    labels.Set(map[string]string{"tier": "critical"}),
)

// Multi-branch logic with Switch
switchTransformer := transformer.Switch(
    []transformer.Case{
        {
            When: namespace.Filter("production"),
            Then: transformer.Chain(
                labels.Set(map[string]string{"env": "prod", "monitoring": "enabled"}),
                annotations.Set(map[string]string{"tier": "critical"}),
            ),
        },
        {
            When: namespace.Filter("staging"),
            Then: labels.Set(map[string]string{"env": "staging"}),
        },
    },
    // Default transformer when no cases match
    labels.Set(map[string]string{"env": "dev"}),
)

e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithTransformer(chainedTransformer),
    engine.WithTransformer(switchTransformer),
)
```

### Built-in Metadata Transformers

```go
// Namespace transformers
setNamespace := nstrans.Set("production")              // Force namespace
ensureDefault := nstrans.EnsureDefault("default")      // Set only if empty

// Name transformers
addPrefix := nametrans.SetPrefix("prod-")
addSuffix := nametrans.SetSuffix("-v2")
replaceName := nametrans.Replace("staging", "production")

// Label transformers
addLabels := labels.Set(map[string]string{
    "env": "prod",
    "team": "platform",
})
removeLabels := labels.Remove("temp", "debug")
removeByPrefix := labels.RemoveIf(func(key, _ string) bool {
    return strings.HasPrefix(key, "temp-")
})

// Annotation transformers
addAnnotations := annotations.Set(map[string]string{
    "rendered-by": "k8s-manifests-lib",
})
removeAnnotations := annotations.Remove("temp-annotation")
removeByValue := annotations.RemoveIf(func(_, value string) bool {
    return value == "delete-me"
})
```

### Real-World Example: Environment-Specific Processing

```go
// Different processing for each environment
filter := filter.And(
    // Exclude system namespaces
    filter.Not(namespace.Filter("kube-system", "kube-public")),
    // Include only Deployments and Services
    filter.Or(
        gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
        gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
    ),
)

transformer := transformer.Switch(
    []transformer.Case{
        {
            // Production: add critical labels and monitoring annotations
            When: namespace.Filter("production"),
            Then: transformer.Chain(
                labels.Set(map[string]string{
                    "env": "prod",
                    "monitoring": "enabled",
                    "backup": "enabled",
                }),
                annotations.Set(map[string]string{
                    "alert-severity": "critical",
                }),
                nametrans.SetPrefix("prod-"),
            ),
        },
        {
            // Staging: basic labels
            When: namespace.Filter("staging"),
            Then: transformer.Chain(
                labels.Set(map[string]string{"env": "staging"}),
                nametrans.SetPrefix("stg-"),
            ),
        },
    },
    // Default for dev
    transformer.Chain(
        labels.Set(map[string]string{"env": "dev"}),
        nametrans.SetPrefix("dev-"),
    ),
)

e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(filter),
    engine.WithTransformer(transformer),
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
