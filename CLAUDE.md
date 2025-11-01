# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

k8s-manifests-lib is a Go library for rendering, filtering, and transforming Kubernetes manifests from multiple sources (Helm, Kustomize, Go templates, YAML). It provides a type-safe, extensible engine with a functional options pattern for configuration.

## Documentation

- **[README.md](README.md)** - Quick start, 5 runnable examples, getting started guide
- **[docs/design.md](docs/design.md)** - Architecture, design decisions, and patterns
- **[docs/development.md](docs/development.md)** - Coding conventions, testing guidelines, and contribution guide
- **[docs/examples.md](docs/examples.md)** - Examples guide (runnable examples and test file usage patterns)

## Quick Reference

### Core Types

- `Renderer`: Interface with `Process(ctx, values map[string]any) ([]unstructured.Unstructured, error)` method
- `Filter`: Function signature `func(ctx, object) (keep bool, err error)`
- `Transformer`: Function signature `func(ctx, object) (transformed object, err error)`

See [docs/design.md](docs/design.md) for complete architecture details.

### Three-Level Pipeline

1. **Renderer-specific**: Filters/transformers applied inside each renderer's `Process()`
2. **Engine-level**: Filters/transformers applied to all renders via `engine.New()`
3. **Render-time**: Filters/transformers applied to a single `Render()` call

See [docs/design.md#8-three-level-filteringtransformation](docs/design.md#8-three-level-filteringtransformation) for details.

### Render-Time Values

Pass dynamic values at render-time to override configuration-time values:

```go
// Render with values that override Source.Values()
objects, err := e.Render(ctx, engine.WithValues(map[string]any{
    "replicaCount": 5,
    "image": map[string]any{"tag": "v2.0"},
}))
```

See [examples/](examples/) for runnable examples and `pkg/**/*_test.go` files for usage patterns.

## Development

For detailed development information:

- **Build and test commands**: See [docs/development.md#setup-and-build](docs/development.md#setup-and-build)
- **Coding conventions**: See [docs/development.md#coding-conventions](docs/development.md#coding-conventions)
- **Testing guidelines**: See [docs/development.md#testing-guidelines](docs/development.md#testing-guidelines)
- **Adding renderers/filters/transformers**: See [docs/development.md#extensibility](docs/development.md#extensibility)
- **Code review guidelines**: See [docs/development.md#code-review-guidelines](docs/development.md#code-review-guidelines)

## Common Tasks

**Run tests:**
```bash
make test
```

**Format and lint:**
```bash
make fmt
make check/lint
```

**Add a new renderer:**
See [docs/development.md#adding-a-new-renderer](docs/development.md#adding-a-new-renderer) for the complete step-by-step guide.

**Add a filter or transformer:**
See [docs/development.md#adding-new-filtertransformer](docs/development.md#adding-new-filtertransformer) for examples and patterns.
