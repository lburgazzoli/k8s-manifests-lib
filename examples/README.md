# k8s-manifests-lib Examples

This directory contains comprehensive examples demonstrating all features of the k8s-manifests-lib library. Examples are organized by complexity and category to help you learn progressively.

## Quick Start

If you're new to the library, start with **[01-basic/helm](01-basic/helm/main.go)** - it's the simplest possible usage.

```bash
go run examples/01-basic/helm/main.go
```

## Running Examples

Each example is a standalone Go program that can be run directly:

```bash
go run examples/<category>/<name>/main.go
```

For example:
```bash
go run examples/02-filtering/namespace/main.go
go run examples/04-composition/transformer-chain/main.go
```

## Categories

### 1. Basic Usage ([01-basic/](01-basic/))

Single-renderer examples using convenience functions. **Start here if you're new.**

| Example | Description | File |
|---------|-------------|------|
| **Helm** | Render Helm chart with `engine.Helm()` | [helm/main.go](01-basic/helm/main.go) |
| **Kustomize** | Render Kustomize directory with `engine.Kustomize()` | [kustomize/main.go](01-basic/kustomize/main.go) |
| **YAML** | Load plain YAML files with `engine.Yaml()` | [yaml/main.go](01-basic/yaml/main.go) |
| **Go Templates** | Render Go templates with `engine.GoTemplate()` | [gotemplate/main.go](01-basic/gotemplate/main.go) |

**Key Concepts**: Convenience functions, basic rendering, single source

### 2. Filtering ([02-filtering/](02-filtering/))

Examples showing different ways to filter Kubernetes objects.

| Example | Description | File |
|---------|-------------|------|
| **Namespace** | Filter by namespace (include/exclude) | [namespace/main.go](02-filtering/namespace/main.go) |
| **Labels** | Filter by labels (HasLabel, MatchLabels, Selector) | [labels/main.go](02-filtering/labels/main.go) |
| **GVK** | Filter by Kind and API version | [gvk/main.go](02-filtering/gvk/main.go) |
| **JQ** | Filter with JQ expressions | [jq/main.go](02-filtering/jq/main.go) |

**Key Concepts**: Filter functions, engine-level filtering, metadata-based selection

### 3. Transformation ([03-transformation/](03-transformation/))

Examples showing how to modify Kubernetes objects.

| Example | Description | File |
|---------|-------------|------|
| **Labels** | Add, update, remove labels (Set, Remove, RemoveIf) | [labels/main.go](03-transformation/labels/main.go) |
| **Annotations** | Add, update, remove annotations | [annotations/main.go](03-transformation/annotations/main.go) |
| **Namespace** | Set or ensure namespace (Set, EnsureDefault) | [namespace/main.go](03-transformation/namespace/main.go) |
| **Name** | Modify object names (Prefix, Suffix, Replace) | [name/main.go](03-transformation/name/main.go) |

**Key Concepts**: Transformer functions, metadata modification, engine-level transformations

### 4. Composition ([04-composition/](04-composition/))

Examples showing how to compose filters and transformers for complex logic.

| Example | Description | File |
|---------|-------------|------|
| **Filter Boolean** | Boolean composition (And, Or, Not) | [filter-boolean/main.go](04-composition/filter-boolean/main.go) |
| **Filter Conditional** | Conditional filtering (If) | [filter-conditional/main.go](04-composition/filter-conditional/main.go) |
| **Transformer Chain** | Sequential transformations (Chain) | [transformer-chain/main.go](04-composition/transformer-chain/main.go) |
| **Transformer Switch** | Multi-branch transformations (Switch) | [transformer-switch/main.go](04-composition/transformer-switch/main.go) |

**Key Concepts**: Composition functions, boolean logic, conditional execution, chaining

### 5. Advanced ([05-advanced/](05-advanced/))

Complex real-world scenarios combining multiple features.

| Example | Description | File |
|---------|-------------|------|
| **Three-Level Pipeline** | Renderer → Engine → Render-time filtering/transformation | [three-level-pipeline/main.go](05-advanced/three-level-pipeline/main.go) |
| **Multi-Environment** | Environment-specific transformations (prod/staging/dev) | [multi-environment/main.go](05-advanced/multi-environment/main.go) |
| **Conditional Transformations** | Apply transformations based on conditions | [conditional-transformations/main.go](05-advanced/conditional-transformations/main.go) |
| **Complex Nested** | Deep nesting of filters and transformers | [complex-nested/main.go](05-advanced/complex-nested/main.go) |

**Key Concepts**: Three-level pipeline, nested composition, real-world patterns

### 6. Renderers ([06-renderers/](06-renderers/))

Advanced renderer usage patterns.

| Example | Description | File |
|---------|-------------|------|
| **Multiple Sources** | Multiple Helm charts in one renderer | [multiple-sources/main.go](06-renderers/multiple-sources/main.go) |
| **Multiple Renderers** | Combining Helm, Kustomize, and YAML | [multiple-renderers/main.go](06-renderers/multiple-renderers/main.go) |
| **Dynamic Values** | Runtime value functions for dynamic configuration | [dynamic-values/main.go](06-renderers/dynamic-values/main.go) |

**Key Concepts**: Multi-source rendering, renderer aggregation, dynamic configuration

### 7. Caching ([07-caching/](07-caching/))

Performance optimization with caching.

| Example | Description | File |
|---------|-------------|------|
| **Basic** | Enable caching with TTL | [basic/main.go](07-caching/basic/main.go) |
| **Performance** | Benchmark cache vs no-cache performance | [performance/main.go](07-caching/performance/main.go) |

**Key Concepts**: Cache configuration, TTL, automatic deep cloning, performance

## Learning Path

We recommend following this learning path:

1. **Start Here**: [01-basic/helm](01-basic/helm/main.go) - Simplest possible usage
2. **Add Filtering**: [02-filtering/namespace](02-filtering/namespace/main.go) - Filter by namespace
3. **Add Transformation**: [03-transformation/labels](03-transformation/labels/main.go) - Modify labels
4. **Compose Logic**: [04-composition/filter-boolean](04-composition/filter-boolean/main.go) - Boolean composition
5. **Advanced Patterns**: [05-advanced/multi-environment](05-advanced/multi-environment/main.go) - Real-world scenario
6. **Optimize**: [07-caching/basic](07-caching/basic/main.go) - Add caching for performance

## Common Patterns

### Pattern 1: Environment-Specific Processing

Filter and transform objects differently for prod/staging/dev environments.

**See**: [05-advanced/multi-environment](05-advanced/multi-environment/main.go)

```go
transformer.Switch(
    []transformer.Case{
        {When: namespace.Filter("production"), Then: prodTransformers},
        {When: namespace.Filter("staging"), Then: stagingTransformers},
    },
    devTransformers, // default
)
```

### Pattern 2: Progressive Filtering

Apply filters at three different levels for fine-grained control.

**See**: [05-advanced/three-level-pipeline](05-advanced/three-level-pipeline/main.go)

```go
// Level 1: Renderer-specific
helm.WithFilter(onlyDeployments)

// Level 2: Engine-level
engine.WithFilter(notSystemNamespaces)

// Level 3: Render-time
e.Render(ctx, engine.WithRenderFilter(productionOnly))
```

### Pattern 3: Conditional Transformations

Apply transformations only when conditions are met.

**See**: [05-advanced/conditional-transformations](05-advanced/conditional-transformations/main.go)

```go
transformer.If(
    namespace.Filter("production"),
    labels.Set(map[string]string{"monitoring": "enabled"}),
)
```

## Tips

- **Start simple**: Begin with basic examples and add complexity gradually
- **Run examples**: Each example is runnable - execute them to see output
- **Read comments**: Examples include inline comments explaining each step
- **Combine concepts**: Advanced examples show how to combine multiple features
- **Check docs**: See [../README.md](../README.md) and [../docs/design.md](../docs/design.md) for detailed documentation

## Troubleshooting

**"Failed to render Helm chart"**
- Ensure you have network access to the OCI registry
- Some examples use public charts that may have rate limits

**"Kustomization directory not found"**
- Some examples reference local directories (e.g., `./kustomization-example`)
- Create the directory or modify the path to match your setup

**"No objects rendered"**
- Check your filters - they might be too restrictive
- Try running without filters first to see all objects

## Contributing Examples

When adding new examples:

1. Create a new directory in the appropriate category
2. Add a standalone `main.go` that can be run directly
3. Include clear comments explaining what's demonstrated
4. Update this README.md with a link to your example
5. Keep examples focused on one concept
6. Make examples self-contained and runnable

## Additional Resources

- **Library Documentation**: [../README.md](../README.md)
- **Design Document**: [../docs/design.md](../docs/design.md)
- **API Reference**: See GoDoc (run `go doc` in package directories)
- **Issue Tracker**: [GitHub Issues](https://github.com/lburgazzoli/k8s-manifests-lib/issues)
