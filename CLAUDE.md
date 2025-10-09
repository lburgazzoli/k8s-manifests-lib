# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

k8s-manifests-lib is a Go library for rendering, filtering, and transforming Kubernetes manifests from multiple sources (Helm, Kustomize, Go templates, YAML). It provides a type-safe, extensible engine with a functional options pattern for configuration.

For detailed architecture and design information, see [docs/design.md](docs/design.md).
For usage examples, see [README.md](README.md).

## Build Commands

```bash
# Run all tests
make test

# Format code
make fmt

# Run linter
make check/lint

# Clean build artifacts and test cache
make clean

# Update dependencies
make deps
```

## Test Commands

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests in a specific package
go test -v ./pkg/renderer/helm

# Run a specific test
go test -v ./pkg/renderer/helm -run TestHelmRenderer

# Run benchmarks
go test -v ./pkg/renderer/... -run=^$ -bench=.
```

## Quick Reference

### Core Types

- `Renderer`: Interface with `Process(ctx) ([]unstructured.Unstructured, error)` method
- `Filter`: Function signature `func(ctx, object) (keep bool, err error)`
- `Transformer`: Function signature `func(ctx, object) (transformed object, err error)`

See [docs/design.md](docs/design.md) for complete architecture details.

### Functional Options Pattern

All configuration uses functional options:

```go
// Function-based
engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithFilter(namespaceFilter),
)

// Struct-based
engine.New(&engine.EngineOptions{
    Renderers: []types.Renderer{helmRenderer},
    Filters: []types.Filter{namespaceFilter},
})
```

See [docs/design.md#151-functional-options-pattern](docs/design.md#151-functional-options-pattern) for implementation guidelines.

### Three-Level Pipeline

1. **Renderer-specific**: Filters/transformers applied inside each renderer's `Process()`
2. **Engine-level**: Filters/transformers applied to all renders via `engine.New()`
3. **Render-time**: Filters/transformers applied to a single `Render()` call

See [docs/design.md#8-three-level-filteringtransformation](docs/design.md#8-three-level-filteringtransformation) for details.

### Caching

Renderers support optional caching with automatic deep cloning:

```go
helm.WithCache(cache.WithTTL(5 * time.Minute))
```

See [docs/design.md#6-caching-architecture](docs/design.md#6-caching-architecture) for implementation details.

## Testing Guidelines

- Use vanilla Gomega (not Ginkgo)
- Use dot imports for Gomega: `import . "github.com/onsi/gomega"`
- Prefer `Should` over `To`
- For error validation: `Should(HaveOccurred())` / `ShouldNot(HaveOccurred())`
- Use subtests (`t.Run`) for organizing related test cases
- Use `t.Context()` instead of `context.Background()` or `context.TODO()` (Go 1.24+)
- Benchmark tests must include renderer name: `BenchmarkHelmRenderWithCache`, `BenchmarkKustomizeRenderCacheMiss`

See [docs/design.md#152-testing-conventions](docs/design.md#152-testing-conventions) for complete testing guidelines.

## Common Patterns

See [README.md](README.md) for complete examples including:

- Basic Helm rendering
- Using cache for performance
- Multiple renderers
- Three-level filtering/transformation
- Dynamic values

## Adding a New Renderer

1. Create package in `pkg/renderer/yourrenderer/`
2. Define a `Source` struct with renderer-specific fields
3. Create `yourrenderer.go` with constructor: `func New(inputs []Source, opts ...RendererOption) *Renderer`
4. Implement `Renderer` interface with `Process(ctx) ([]unstructured.Unstructured, error)` method
5. Create `yourrenderer_option.go` with functional options following the pattern in `pkg/util/option.go`
6. In `Process()`, apply renderer-specific filters/transformers using `pipeline.ApplyFilters()` and `pipeline.ApplyTransformers()`

All renderers follow the consistent `[]Source` pattern for type-safe, flexible input handling.

See [docs/design.md#131-adding-a-new-renderer](docs/design.md#131-adding-a-new-renderer) for complete details.
