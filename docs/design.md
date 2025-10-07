# Design Document: k8s-manifests-lib

## 1. Introduction

This document outlines the design of `k8s-manifests-lib`, a Go library for rendering Kubernetes manifests from various sources into `unstructured.Unstructured` objects. The library provides a robust and extensible engine with compile-time type safety for renderer inputs and a function-based approach for filters and transformers.

The primary goal is to offer comprehensive capabilities for rendering, filtering, and transforming Kubernetes resources from multiple sources (Helm, Kustomize, Go templates, YAML files, and in-memory objects).

## 2. High-Level Architecture

The library consists of a central `Engine` that orchestrates the rendering process. The `Engine` is configured with `Renderer` instances, each responsible for rendering manifests from a specific source. The rendering pipeline has three distinct stages for filters and transformers:

1. **Renderer-specific**: Applied within each renderer's `Process()` method
2. **Engine-level**: Applied to aggregated results from all renderers
3. **Render-time**: Applied to a specific `Render()` call, merged with engine-level filters/transformers

```
┌────────────────────┐
│ Engine             │
│ Configuration      │
└──────────┬─────────┘
           │
           ├──► Renderer 1 ──► Process + Renderer F/T ──┐
           │                                            │
           ├──► Renderer 2 ──► Process + Renderer F/T ──┼──► Aggregate
           │                                            │    Objects
           └──► Renderer N ──► Process + Renderer F/T ──┘
                                                         │
                                                         ▼
                                              Engine-Level Filters
                                                         │
                                                         ▼
                                           Engine-Level Transformers
                                                         │
                                                         ▼
                                              Render-Time Filters
                                                         │
                                                         ▼
                                           Render-Time Transformers
                                                         │
                                                         ▼
                                                   Final Objects
```

## 3. Core Concepts

### 3.1. Package Structure

```
k8s-manifests-lib/
├── pkg/
│   ├── types/           # Core type definitions
│   │   └── types.go     # Renderer, Filter, Transformer
│   ├── engine/          # Engine implementation
│   │   ├── engine.go
│   │   └── engine_option.go
│   ├── renderer/        # Renderer implementations
│   │   ├── helm/
│   │   ├── kustomize/
│   │   ├── gotemplate/
│   │   ├── yaml/
│   │   └── mem/
│   ├── filter/          # Filter implementations and composition
│   │   ├── compose.go   # Filter composition (Or, And, Not, If)
│   │   ├── jq/
│   │   └── meta/
│   │       ├── annotations/  # Annotation filters
│   │       ├── gvk/         # GroupVersionKind filters
│   │       ├── labels/      # Label filters
│   │       ├── name/        # Name filters
│   │       └── namespace/   # Namespace filters
│   ├── transformer/     # Transformer implementations and composition
│   │   ├── compose.go   # Transformer composition (Chain, If, Switch)
│   │   ├── jq/
│   │   └── meta/
│   │       ├── annotations/  # Annotation transformers
│   │       ├── labels/       # Label transformers
│   │       ├── name/         # Name transformers
│   │       └── namespace/    # Namespace transformers
│   └── util/           # Utility functions
│       ├── filter.go
│       ├── yaml.go
│       ├── option.go
│       └── cache/      # Caching implementation
│           ├── cache.go
│           └── cache_option.go
```

### 3.2. Core Types (pkg/types/types.go)

```go
// Renderer is the interface that all concrete renderers implement.
type Renderer interface {
    Process(ctx context.Context) ([]unstructured.Unstructured, error)
}

// Filter is a function that decides whether to keep an object.
type Filter func(ctx context.Context, object unstructured.Unstructured) (bool, error)

// Transformer is a function that transforms an object.
type Transformer func(ctx context.Context, object unstructured.Unstructured) (unstructured.Unstructured, error)
```

### 3.3. Engine (pkg/engine/engine.go)

The `Engine` struct manages the rendering pipeline:

```go
type Engine struct {
    options engineOptions
}

// New creates a new Engine with the given options.
func New(opts ...EngineOption) *Engine

// Render processes all registered renderers and applies filters/transformers.
func (e *Engine) Render(ctx context.Context, opts ...RenderOption) ([]unstructured.Unstructured, error)
```

**Rendering Pipeline:**

1. Process each renderer sequentially via `renderer.Process(ctx)`
2. Aggregate all objects from all renderers
3. Apply engine-level filters (configured via `New()`)
4. Apply engine-level transformers (configured via `New()`)
5. Apply render-time filters (passed to `Render()`)
6. Apply render-time transformers (passed to `Render()`)

## 4. Configuration Pattern

The library uses the **functional options pattern** with dual support:

1. **Function-based options**: `WithRenderer(r)`, `WithFilter(f)`, `WithTransformer(t)`
2. **Struct-based options**: Direct struct literals for bulk configuration

All options implement the `Option[T]` interface:

```go
type Option[T any] interface {
    ApplyTo(target *T)
}
```

### 4.1. Engine Options

```go
// Engine configuration
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(appsV1Filter),
    engine.WithTransformer(labelTransformer),
)

// Or using struct-based options
e := engine.New(&engine.EngineOptions{
    Renderers: []types.Renderer{helmRenderer},
    Filters: []types.Filter{appsV1Filter},
    Transformers: []types.Transformer{labelTransformer},
})
```

### 4.2. Render-Time Options

```go
// Function-based
objects, err := e.Render(ctx,
    engine.WithRenderFilter(namespaceFilter),
    engine.WithRenderTransformer(annotationTransformer),
)

// Struct-based
objects, err := e.Render(ctx, engine.RenderOptions{
    Filters: []types.Filter{namespaceFilter},
    Transformers: []types.Transformer{annotationTransformer},
})
```

## 5. Renderer Implementations

All renderers follow a consistent pattern:

1. Define a `Source` struct for typed inputs
2. Implement the `types.Renderer` interface
3. Provide a constructor: `New(inputs []Source, opts ...RendererOption) (*Renderer, error)`
4. Support renderer-specific filters and transformers via options
5. Optional caching support via `WithCache(opts...)`

### 5.0. Engine Convenience Functions

For single-renderer scenarios, the `engine` package provides convenience factory functions:

```go
// Instead of creating renderer then engine:
helmRenderer, _ := helm.New([]helm.Source{{...}})
e := engine.New(engine.WithRenderer(helmRenderer))

// Use convenience function directly (takes single Source):
e, _ := engine.Helm(helm.Source{...})
```

Available factory functions in `pkg/engine/engine_support.go`:
* `engine.Helm(source, opts...)` - Creates Engine with single Helm renderer
* `engine.Kustomize(source, opts...)` - Creates Engine with single Kustomize renderer
* `engine.Yaml(source, opts...)` - Creates Engine with single YAML renderer
* `engine.GoTemplate(source, opts...)` - Creates Engine with single Go template renderer
* `engine.Mem(source, opts...)` - Creates Engine with single memory renderer

**When to use:**
* **Convenience functions**: Single renderer, simple use cases
* **Full Engine API**: Multiple renderers, engine-level filters/transformers, complex pipelines

### 5.1. Helm (pkg/renderer/helm)

Renders Helm charts from OCI registries, HTTP repositories, or local paths.

```go
type Source struct {
    Repo                string                                         // Repository URL (optional)
    Chart               string                                         // Chart name or path (required)
    ReleaseName         string                                         // Release name (required)
    ReleaseVersion      string                                         // Chart version (optional)
    Values              func(context.Context) (map[string]any, error)  // Dynamic values function
    ProcessDependencies bool                                           // Process chart dependencies
}

// Constructor
func New(inputs []Source, opts ...RendererOption) (*Renderer, error)

// Options
helm.WithFilter(filter)
helm.WithTransformer(transformer)
helm.WithCache(cache.WithTTL(5 * time.Minute))  // Enable caching
```

**Features:**

* OCI registry support: `oci://registry-1.docker.io/org/chart`
* HTTP repository support: `https://charts.example.com`
* Dynamic values via `ValuesFunc`
* Specific chart versions via `ReleaseVersion`
* Optional caching for improved performance

### 5.2. Kustomize (pkg/renderer/kustomize)

Renders Kustomize overlays using the official Kustomize API.

```go
type Source struct {
    Path   string                                     // Path to kustomization directory (required)
    Values func(context.Context) (map[string]string, error)  // Dynamic values as ConfigMap
}

func New(inputs []Source, opts ...RendererOption) (*Renderer, error)
```

### 5.3. Go Template (pkg/renderer/gotemplate)

Renders Go templates with `fs.FS` support.

```go
type Source struct {
    FS     fs.FS                                // Filesystem containing templates (required)
    Path   string                               // Glob pattern for templates (required)
    Values func(context.Context) (any, error)  // Dynamic template values
}

func New(inputs []Source, opts ...RendererOption) (*Renderer, error)
```

**Features:**

* Embedded filesystem support via `fs.FS`
* Glob pattern matching for templates
* Dynamic values via `ValuesFunc`
* Optional caching based on values hash

### 5.4. YAML (pkg/renderer/yaml)

Loads plain YAML files with `fs.FS` and glob support.

```go
type Source struct {
    FS   fs.FS  // Filesystem containing YAML files (required)
    Path string // Glob pattern for YAML files (required)
}

func New(inputs []Source, opts ...RendererOption) (*Renderer, error)
```

**Features:**

* Multi-document YAML support
* Glob pattern matching
* Both `.yaml` and `.yml` extensions
* Optional caching based on file path

### 5.5. Memory (pkg/renderer/mem)

In-memory renderer for testing or passthrough scenarios.

```go
type Source struct {
    Objects []unstructured.Unstructured
}

func New(inputs []Source, opts ...RendererOption) (*Renderer, error)
```

## 6. Caching Architecture

### 6.1. Overview

The library provides a custom caching implementation to improve rendering performance. The cache was designed to:

* Reduce external dependencies (replaced `k8s.io/client-go/tools/cache`)
* Provide TTL-based expiration with lazy cleanup
* Prevent cache pollution through automatic deep cloning
* Support generic types for flexibility

### 6.2. Cache Interface

```go
// Generic cache interface
type Interface[T any] interface {
    Get(key string) (T, bool)
    Set(key string, value T)
    Sync()  // Triggers lazy expiration of TTL'd entries
}
```

### 6.3. Implementations

**Private `defaultCache[T]`**: Generic TTL-based cache

```go
type defaultCache[T any] struct {
    mu      sync.RWMutex
    entries map[string]entry[T]
    ttl     time.Duration
}

type entry[T any] struct {
    value     T
    expiresAt time.Time
}
```

**Private `renderCache`**: Wrapper for rendering with automatic deep cloning

```go
type renderCache struct {
    cache Interface[[]unstructured.Unstructured]
}

// Automatically clones on Get to prevent external modifications from affecting cache
func (c *renderCache) Get(key string) ([]unstructured.Unstructured, bool) {
    if value, found := c.cache.Get(key); found {
        return k8s.DeepCloneUnstructuredSlice(value), true
    }
    return nil, false
}

// Automatically clones on Set to prevent caller modifications from affecting cache
func (c *renderCache) Set(key string, value []unstructured.Unstructured) {
    c.cache.Set(key, k8s.DeepCloneUnstructuredSlice(value))
}
```

### 6.4. Public Constructors

```go
// Create a generic cache with TTL
func New[T any](opts ...Option) Interface[T]

// Create a render-specific cache with automatic deep cloning
func NewRenderCache(opts ...Option) Interface[[]unstructured.Unstructured]
```

### 6.5. Configuration

```go
// Configure TTL (defaults to 5 minutes if not specified or invalid)
cache.WithTTL(10 * time.Minute)

// Usage in renderer
helmRenderer, _ := helm.New(
    []helm.Source{{...}},
    helm.WithCache(cache.WithTTL(5 * time.Minute)),
)
```

### 6.6. Cache Behavior

**TTL Expiration:**
* Entries are marked with expiration time on `Set()`
* Expiration is checked lazily on `Get()` - expired entries return as "not found"
* `Sync()` actively removes expired entries from storage

**Deep Cloning:**
* `renderCache` automatically clones on both `Get()` and `Set()`
* Prevents cache pollution from external modifications
* Caller can safely modify returned objects without affecting cache

**Cache Keys:**
* Helm: Hash of render values
* Kustomize: Hash of path + values
* GoTemplate: Hash of template values
* YAML: File path pattern

### 6.7. Benefits

1. **Reduced Dependencies**: No longer depends on `k8s.io/client-go/tools/cache`
2. **Type Safety**: Generic interface allows compile-time type checking
3. **Automatic Safety**: Deep cloning prevents accidental cache pollution
4. **Performance**: Lazy expiration avoids background goroutines
5. **Flexibility**: Works with any type via `Interface[T]`

## 7. Filters and Transformers

Filters and transformers are implemented as constructor functions that return `types.Filter` or `types.Transformer` closures. The library provides composition functions for combining filters and transformers, as well as built-in implementations for common metadata operations.

### 7.1. Filter Composition (pkg/filter)

Combinators for building complex filter logic from simple filters.

```go
// Boolean Logic
func Or(filters ...types.Filter) types.Filter   // Any filter must pass
func And(filters ...types.Filter) types.Filter  // All filters must pass
func Not(filter types.Filter) types.Filter      // Inverts filter result

// Conditional
func If(condition types.Filter, then types.Filter) types.Filter  // Apply 'then' only if condition passes

// Usage: Complex namespace and kind filtering
filter := filter.Or(
    filter.And(
        namespace.Filter("production"),
        gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment")),
    ),
    filter.And(
        namespace.Filter("staging"),
        gvk.Filter(corev1.SchemeGroupVersion.WithKind("Service")),
    ),
)
```

**Composition Features:**
* Short-circuit evaluation for performance
* Arbitrary nesting depth
* Clear, readable filter logic
* Composable with all filter types

### 7.2. Transformer Composition (pkg/transformer)

Combinators for building complex transformation pipelines.

```go
// Sequential Execution
func Chain(transformers ...types.Transformer) types.Transformer  // Apply transformers in sequence

// Conditional Transformation
func If(condition types.Filter, transformer types.Transformer) types.Transformer  // Apply only if condition passes

// Multi-branch Logic
type Case struct {
    When types.Filter
    Then types.Transformer
}
func Switch(cases []Case, defaultTransformer types.Transformer) types.Transformer  // First matching case wins

// Usage: Environment-specific transformations
transformer := transformer.Switch(
    []transformer.Case{
        {
            When: namespace.Filter("production"),
            Then: transformer.Chain(
                labels.Set(map[string]string{"env": "prod"}),
                annotations.Set(map[string]string{"tier": "critical"}),
            ),
        },
        {
            When: namespace.Filter("staging"),
            Then: labels.Set(map[string]string{"env": "staging"}),
        },
    },
    labels.Set(map[string]string{"env": "dev"}), // default
)
```

**Composition Features:**
* Lazy evaluation - transformers only execute when conditions match
* Early exit in Switch - first matching case wins
* Composable with all transformer types
* Type-safe Case definitions

### 7.3. Namespace Filters (pkg/filter/meta/namespace)

```go
// Constructors
func Filter(namespaces ...string) types.Filter  // Include only these namespaces
func Exclude(namespaces ...string) types.Filter // Exclude these namespaces

// Usage
includeFilter := namespace.Filter("production", "staging")
excludeFilter := namespace.Exclude("kube-system", "kube-public")
```

### 7.4. Label Filters (pkg/filter/meta/labels)

```go
// Constructors
func HasLabel(key string) types.Filter                          // Has specific label key
func HasLabels(keys ...string) types.Filter                     // Has all specified keys
func MatchLabels(matchLabels map[string]string) types.Filter    // All labels match exactly
func Selector(selector string) (types.Filter, error)            // Kubernetes label selector syntax

// Usage
hasEnvLabel := labels.HasLabel("environment")
matchProd := labels.MatchLabels(map[string]string{"env": "prod", "tier": "frontend"})
selectorFilter, _ := labels.Selector("app=nginx,tier in (frontend,backend)")
```

### 7.5. Name Filters (pkg/filter/meta/name)

```go
// Constructors
func Exact(names ...string) types.Filter        // Exact name match
func Prefix(prefix string) types.Filter          // Name starts with prefix
func Suffix(suffix string) types.Filter          // Name ends with suffix
func Regex(pattern string) (types.Filter, error) // Name matches regex pattern

// Usage
exactFilter := name.Exact("nginx-deployment", "redis-service")
prefixFilter := name.Prefix("app-")
regexFilter, _ := name.Regex(`^(nginx|apache)-.*$`)
```

### 7.6. Annotation Filters (pkg/filter/meta/annotations)

```go
// Constructors
func HasAnnotation(key string) types.Filter                             // Has specific annotation key
func HasAnnotations(keys ...string) types.Filter                        // Has all specified keys
func MatchAnnotations(matchAnnotations map[string]string) types.Filter  // All annotations match exactly

// Usage
hasOwner := annotations.HasAnnotation("owner")
matchFilter := annotations.MatchAnnotations(map[string]string{
    "managed-by": "k8s-manifests-lib",
})
```

### 7.7. Namespace Transformers (pkg/transformer/meta/namespace)

```go
// Constructors
func Set(namespace string) types.Transformer            // Set namespace unconditionally
func EnsureDefault(namespace string) types.Transformer  // Set only if empty

// Usage
forceNamespace := namespacetrans.Set("production")
defaultNamespace := namespacetrans.EnsureDefault("default")
```

### 7.8. Name Transformers (pkg/transformer/meta/name)

```go
// Constructors
func SetPrefix(prefix string) types.Transformer             // Add prefix to name
func SetSuffix(suffix string) types.Transformer             // Add suffix to name
func Replace(from string, to string) types.Transformer      // Replace substring in name

// Usage
addPrefix := nametrans.SetPrefix("prod-")
addSuffix := nametrans.SetSuffix("-v2")
replaceEnv := nametrans.Replace("staging", "production")
```

### 7.9. Label Transformers (pkg/transformer/meta/labels)

```go
// Constructors
func Transform(labels map[string]string) types.Transformer                        // Add/update labels
func Remove(keys ...string) types.Transformer                                     // Remove specific labels
func RemoveIf(predicate func(key string, value string) bool) types.Transformer    // Remove matching labels

// Usage
addLabels := labels.Set(map[string]string{"env": "prod", "team": "platform"})
removeLabels := labels.Remove("temp", "debug")
removePrefix := labels.RemoveIf(func(key, _ string) bool {
    return strings.HasPrefix(key, "temp-")
})
```

### 7.10. Annotation Transformers (pkg/transformer/meta/annotations)

```go
// Constructors
func Transform(annotations map[string]string) types.Transformer                   // Add/update annotations
func Remove(keys ...string) types.Transformer                                     // Remove specific annotations
func RemoveIf(predicate func(key string, value string) bool) types.Transformer    // Remove matching annotations

// Usage
addAnnotations := annotations.Set(map[string]string{
    "rendered-by": "k8s-manifests-lib",
})
removeAnnotations := annotations.Remove("temp-annotation")
```

### 7.11. JQ Filter (pkg/filter/jq)

```go
// Constructor
func Filter(expression string, opts ...Option) (types.Filter, error)

// Usage
filter, err := jq.Filter(`.kind == "Deployment"`)

// With variables
filter, err := jq.Filter(
    `.kind == $expectedKind`,
    jq.WithVariable("expectedKind", "Pod"),
)
```

### 7.12. GVK Filter (pkg/filter/meta/gvk)

```go
// Constructor
func Filter(gvks ...schema.GroupVersionKind) types.Filter

// Usage
filter := gvk.Filter(
    corev1.SchemeGroupVersion.WithKind("Pod"),
    corev1.SchemeGroupVersion.WithKind("Service"),
)
```

### 7.13. JQ Transformer (pkg/transformer/jq)

```go
// Constructor
func Transform(expression string, opts ...Option) (types.Transformer, error)

// Usage
transformer, err := jq.Transform(`. + {"metadata": {"labels": {"new": "label"}}}`)
```

## 8. Three-Level Filtering/Transformation

The library supports filtering and transformation at three distinct stages:

### 8.1. Renderer-Specific (Earliest)

Applied by individual renderers during their `Process()` method, before results are returned.

```go
helmRenderer, err := helm.New(
    []helm.Source{...},
    helm.WithFilter(onlyDeploymentsFilter),         // Applied by Helm only
    helm.WithTransformer(addHelmLabelsTransformer), // Applied by Helm only
)
```

**Use when**: You want filtering/transformation specific to one renderer's output.

### 8.2. Engine-Level (Middle)

Applied to aggregated results from all renderers on every `Render()` call.

```go
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(namespaceFilter),      // Applied to ALL renders
    engine.WithTransformer(addCommonLabels), // Applied to ALL renders
)
```

**Use when**: You want consistent filtering/transformation across all renders.

### 8.3. Render-Time (Latest)

Applied to a single `Render()` call, merged with engine-level filters/transformers.

```go
objects, err := e.Render(ctx,
    engine.WithRenderFilter(kindFilter),               // Applied only to this render
    engine.WithRenderTransformer(envLabelTransformer), // Applied only to this render
)
```

**Use when**: You need one-off filtering/transformation for a specific operation.

**Important**: Render-time options are *additive* - they append to engine-level options.

### 8.4. Execution Order

```
1. Renderer processes inputs + applies renderer-specific F/T
2. Engine aggregates all renderer results
3. Engine applies engine-level filters
4. Engine applies render-time filters (merged)
5. Engine applies engine-level transformers
6. Engine applies render-time transformers (merged)
7. Returns final objects
```

## 9. Filter and Transformer Logic

### 9.1. Filter Logic (AND Semantics)

Multiple filters are combined with **AND logic** - an object must pass ALL filters to be kept.

```go
engine.New(
    engine.WithFilter(namespaceFilter),  // Must pass this
    engine.WithFilter(kindFilter),        // AND must pass this
)
```

Implementation in `util.ApplyFilters()` returns false as soon as any filter rejects an object.

### 9.2. Transformer Chaining

Transformers are applied **sequentially** - the output of one becomes the input to the next.

```go
engine.New(
    engine.WithTransformer(labels.Set(map[string]string{"env": "prod"})),
    engine.WithTransformer(annotations.Set(map[string]string{"version": "1.0"})),
)
```

**Order matters!** Implementation in `util.ApplyTransformers()` processes transformers in sequence.

## 10. Utility Functions (pkg/util)

### 10.1. Filter/Transformer Application

```go
// Apply filters with AND logic
func ApplyFilters(ctx context.Context, objects []unstructured.Unstructured, filters []types.Filter) ([]unstructured.Unstructured, error)

// Apply transformers in sequence
func ApplyTransformers(ctx context.Context, objects []unstructured.Unstructured, transformers []types.Transformer) ([]unstructured.Unstructured, error)
```

### 10.2. YAML Decoding

```go
// Decode multi-document YAML into unstructured objects
func DecodeYAML(decoder runtime.Decoder, content []byte) ([]unstructured.Unstructured, error)
```

Handles multi-document YAML streams and skips empty documents.

## 11. Error Handling

* Errors are wrapped using `fmt.Errorf` with `%w` for proper error chain propagation
* Context is passed through the entire pipeline for cancellation support
* First error encountered stops processing and is returned immediately
* All renderer constructors validate inputs and return errors

## 12. Usage Examples

### 12.1. Basic Rendering with Convenience Functions

For single-renderer scenarios, use convenience factory functions:

```go
// Using convenience function - simplest approach
e, err := engine.Helm(helm.Source{
    Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
    ReleaseName: "my-release",
    Values: helm.Values(map[string]any{
        "shared": map[string]any{"appId": "test-app"},
    }),
})
if err != nil {
    log.Fatal(err)
}

objects, err := e.Render(context.Background())
```

Available convenience functions:
* `engine.Helm()` - For Helm charts
* `engine.Kustomize()` - For Kustomize directories
* `engine.Yaml()` - For YAML files
* `engine.GoTemplate()` - For Go templates
* `engine.Mem()` - For in-memory objects

### 12.1b. Basic Rendering with Full Engine API

For more control, use the full Engine API:

```go
helmRenderer, err := helm.New([]helm.Source{
    {
        Chart:       "oci://registry-1.docker.io/daprio/dapr-shared-chart",
        ReleaseName: "my-release",
        Values: helm.Values(map[string]any{
            "shared": map[string]any{"appId": "test-app"},
        }),
    },
})
if err != nil {
    log.Fatal(err)
}

e := engine.New(engine.WithRenderer(helmRenderer))
objects, err := e.Render(context.Background())
```

### 12.2. Rendering with Cache

```go
helmRenderer, err := helm.New(
    []helm.Source{{
        Chart:       "oci://registry-1.docker.io/my-chart",
        ReleaseName: "cached-release",
        Values: helm.Values(map[string]any{"replicaCount": 3}),
    }},
    helm.WithCache(cache.WithTTL(5 * time.Minute)),  // Enable 5-minute cache
)

e := engine.New(engine.WithRenderer(helmRenderer))

// First render: cache miss
objects1, _ := e.Render(context.Background())

// Second render: cache hit (automatic deep clone)
objects2, _ := e.Render(context.Background())

// Modifications don't affect cache
objects2[0].SetName("modified")
```

### 12.3. Three-Level Filtering Example

```go
// 1. Renderer-specific: Only Deployments from Helm
deploymentFilter := gvk.Filter(appsv1.SchemeGroupVersion.WithKind("Deployment"))
helmRenderer, _ := helm.New(
    []helm.Source{...},
    helm.WithFilter(deploymentFilter),
)

// 2. Engine-level: Add common labels to everything
commonLabels := labels.Set(map[string]string{"managed-by": "k8s-manifests-lib"})
e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithTransformer(commonLabels),
)

// 3. Render-time: Filter by namespace for this render only
namespaceFilter, _ := jq.Filter(`.metadata.namespace == "production"`)
objects, err := e.Render(ctx,
    engine.WithRenderFilter(namespaceFilter),
)
```

### 12.4. Multiple Renderers

```go
helmRenderer, _ := helm.New([]helm.Source{...})
kustomizeRenderer, _ := kustomize.New([]kustomize.Source{{Path: "/path/to/base"}})

e := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithRenderer(kustomizeRenderer),
)

objects, err := e.Render(ctx)
// Contains objects from both Helm and Kustomize
```

### 12.5. Dynamic Values

```go
e, _ := engine.Helm(helm.Source{
    Chart:       "oci://registry/chart",
    ReleaseName: "dynamic",
    Values: func(ctx context.Context) (map[string]any, error) {
        return map[string]any{
            "appId": xid.New().String(),
            "timestamp": time.Now().Unix(),
        }, nil
    },
})
```

## 13. Extensibility

### 13.1. Adding a New Renderer

1. Create package in `pkg/renderer/yourrenderer/`
2. Define `Source` struct with renderer-specific fields
3. Create constructor: `func New(inputs []Source, opts ...RendererOption) (*Renderer, error)`
4. Implement `types.Renderer` interface with `Process(ctx)` method
5. Create `yourrenderer_option.go` following the pattern in `pkg/util/option.go`
6. In `Process()`, iterate inputs, render each, apply renderer-specific F/T using `util.ApplyFilters()` and `util.ApplyTransformers()`

### 13.2. Adding New Filter/Transformer

1. Define a constructor function that returns `types.Filter` or `types.Transformer`
2. If configuration is needed, accept parameters and return a closure
3. Add via `engine.WithFilter`/`engine.WithTransformer` for engine-level or via renderer options

Example:

```go
// pkg/filter/custom/custom.go
func MyCustomFilter(threshold int) types.Filter {
    return func(ctx context.Context, obj unstructured.Unstructured) (bool, error) {
        // Custom logic using threshold
        return true, nil
    }
}

// Usage
filter := custom.MyCustomFilter(10)
e := engine.New(engine.WithFilter(filter))
```

## 14. Testing Strategy

* **Unit Tests**: Test each component in isolation
  - Renderers: Test `Process()` with various inputs
  - Filters: Test with matching and non-matching objects
  - Transformers: Verify transformations are applied correctly
  - Cache: Test TTL expiration, deep cloning, Get/Set behavior

* **Integration Tests**: Test the full pipeline
  - Multiple renderers with engine-level F/T
  - Render-time options merging with engine-level
  - Error handling throughout the pipeline
  - Cache integration with renderers

* **Benchmark Tests**: Performance testing
  - Named with renderer prefix: `BenchmarkHelmRenderWithCache`, `BenchmarkKustomizeRenderCacheMiss`
  - Test cache hit vs miss performance
  - Measure deep cloning overhead

* **Test Patterns**:
  - Use vanilla Gomega (no Ginkgo)
  - Subtests via `t.Run()`
  - Use `t.Context()` instead of `context.Background()`
  - Mock renderers for engine tests to avoid external dependencies

## 15. Coding Conventions

### 15.1. Functional Options Pattern

All struct initialization uses the functional options pattern for flexible, extensible configuration:

**Define Options as Interfaces:**
```go
type Option[T any] interface {
    ApplyTo(target *T)
}
```

**Provide Both Function-Based and Struct-Based Options:**
```go
// Function-based option
func WithRenderer(r types.Renderer) EngineOption {
    return util.FunctionalOption[Engine](func(e *Engine) {
        e.renderers = append(e.renderers, r)
    })
}

// Struct-based option for bulk configuration
type EngineOptions struct {
    Renderers    []types.Renderer
    Filters      []types.Filter
    Transformers []types.Transformer
}

func (opts EngineOptions) ApplyTo(e *Engine) {
    e.renderers = opts.Renderers
    e.filters = opts.Filters
    e.transformers = opts.Transformers
}
```

**Guidelines:**
- For slice/map fields in struct-based options, use the type directly (not pointers)
- Place all options and related methods in `*_option.go` files
- Provide both patterns to support different use cases

**Usage:**
```go
// Function-based (flexible, composable)
engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(filter),
)

// Struct-based (bulk configuration via literals)
engine.New(&engine.EngineOptions{
    Renderers: []types.Renderer{helmRenderer},
    Filters:   []types.Filter{filter},
})
```

### 15.2. Testing Conventions

**Framework:**
- Use vanilla Gomega (not Ginkgo)
- Use dot imports for Gomega packages: `import . "github.com/onsi/gomega"`

**Assertions:**
- Prefer `Should` over `To`
- For error validation: `Should(HaveOccurred())` / `ShouldNot(HaveOccurred())`

**Test Structure:**
- Use subtests via `t.Run()` for organizing related test cases
- Use `t.Context()` instead of `context.Background()` or `context.TODO()` (Go 1.24+)

**Test Data:**
- When creating test structs, use Gomega matchers to validate results
- Avoid passing expected values or booleans in test structs

**Benchmark Naming:**
- Include renderer name in benchmark tests
- Format: `Benchmark<Renderer><TestName>`
- Examples: `BenchmarkHelmRenderWithCache`, `BenchmarkKustomizeRenderCacheMiss`

Example:
```go
func TestRenderer(t *testing.T) {
    g := NewWithT(t)
    ctx := t.Context()

    t.Run("should render correctly", func(t *testing.T) {
        result, err := renderer.Process(ctx)
        g.Expect(err).ShouldNot(HaveOccurred())
        g.Expect(result).Should(HaveLen(3))
    })
}
```

## 16. Design Principles

1. **Type Safety**: Compile-time type safety for renderer inputs via typed `Source` structs
2. **Modularity**: Each renderer is independent and self-contained
3. **Flexibility**: Three-level F/T allows precise control over processing
4. **Consistency**: All renderers follow the same pattern and interface
5. **Extensibility**: Easy to add new renderers, filters, and transformers
6. **Error Handling**: Explicit error handling with wrapped errors for debugging
7. **Context Propagation**: Full support for cancellation and timeouts
8. **Functional Options**: Dual pattern support (function-based and struct-based)
9. **Performance**: Optional caching with automatic safety via deep cloning
10. **Minimal Dependencies**: Custom implementations to reduce external dependencies
