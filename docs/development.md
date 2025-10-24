# Development Guide: k8s-manifests-lib

This document provides coding conventions, testing guidelines, and contribution practices for developing k8s-manifests-lib.

For architectural information and design decisions, see [design.md](design.md).
For quick start and usage examples, see [../README.md](../README.md).

## Table of Contents

1. [Setup and Build](#setup-and-build)
2. [Coding Conventions](#coding-conventions)
3. [Testing Guidelines](#testing-guidelines)
4. [Extensibility](#extensibility)
5. [Code Review Guidelines](#code-review-guidelines)

## Setup and Build

### Build Commands

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

### Test Commands

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

## Coding Conventions

### Functional Options Pattern

All struct initialization uses the functional options pattern for flexible, extensible configuration.

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

### Error Handling Conventions

* Errors are wrapped using `fmt.Errorf` with `%w` for proper error chain propagation
* Context is passed through the entire pipeline for cancellation support
* First error encountered stops processing and is returned immediately
* All renderer constructors validate inputs and return errors
* Use `errors.As()` to extract typed errors from error chains
* Use `errors.Is()` to check for specific underlying errors

### Package Organization

```
k8s-manifests-lib/
├── pkg/
│   ├── types/           # Core type definitions
│   ├── engine/          # Engine implementation
│   ├── renderer/        # Renderer implementations
│   │   ├── helm/
│   │   ├── kustomize/
│   │   ├── gotemplate/
│   │   ├── yaml/
│   │   └── mem/
│   ├── filter/          # Filter implementations and composition
│   ├── transformer/     # Transformer implementations and composition
│   ├── pipeline/        # Pipeline execution
│   └── util/            # Utility functions
```

Each renderer follows the pattern:
- `renderer.go` - Main implementation
- `renderer_option.go` - Functional options
- `renderer_test.go` - Tests
- `renderer_support.go` - Helper functions (if needed)

## Testing Guidelines

### Test Framework

- Use vanilla Gomega (not Ginkgo)
- Use dot imports for Gomega: `import . "github.com/onsi/gomega"`
- Prefer `Should` over `To`
- For error validation: `Should(HaveOccurred())` / `ShouldNot(HaveOccurred())`
- Use subtests (`t.Run`) for organizing related test cases
- Use `t.Context()` instead of `context.Background()` or `context.TODO()` (Go 1.24+)

**Example:**
```go
func TestRenderer(t *testing.T) {
    g := NewWithT(t)
    ctx := t.Context()

    t.Run("should render correctly", func(t *testing.T) {
        result, err := renderer.Process(ctx, nil)
        g.Expect(err).ShouldNot(HaveOccurred())
        g.Expect(result).Should(HaveLen(3))
    })
}
```

### Test Data Organization

**CRITICAL**: All test data must be defined as package-level constants, never inline within test methods.

**Good:**
```go
const testKustomization = `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
`

func TestSomething(t *testing.T) {
    writeFile(t, dir, "kustomization.yaml", testKustomization)
    // ...
}
```

**Bad:**
```go
func TestSomething(t *testing.T) {
    kustomization := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
`  // WRONG: inline test data
    writeFile(t, dir, "kustomization.yaml", kustomization)
    // ...
}
```

**Rules:**
- ALL test data (YAML, JSON, strings, etc.) must be package-level constants
- Define constants at the top of test files, grouped by test scenario
- Use descriptive names that indicate purpose (e.g., `nestedResourcesKustomization`, `annotationsBaseConfigMap`)
- Add comments to group related constants (e.g., `// Test constants for nested resources test`)
- This makes tests more readable and data reusable across tests

### Benchmark Naming

- Include renderer name in benchmark tests
- Format: `Benchmark<Renderer><TestName>`
- Examples: `BenchmarkHelmRenderWithCache`, `BenchmarkKustomizeRenderCacheMiss`

### Test Strategy

**Unit Tests**: Test each component in isolation
- Renderers: Test `Process()` with various inputs
- Filters: Test with matching and non-matching objects
- Transformers: Verify transformations are applied correctly
- Cache: Test TTL expiration, deep cloning, Get/Set behavior

**Integration Tests**: Test the full pipeline
- Multiple renderers with engine-level F/T
- Render-time options merging with engine-level
- Error handling throughout the pipeline
- Cache integration with renderers

**Benchmark Tests**: Performance testing
- Named with renderer prefix: `BenchmarkHelmRenderWithCache`, `BenchmarkKustomizeRenderCacheMiss`
- Test cache hit vs miss performance
- Measure deep cloning overhead

**Test Patterns**:
- Use vanilla Gomega (no Ginkgo)
- Subtests via `t.Run()`
- Use `t.Context()` instead of `context.Background()`
- Mock renderers for engine tests to avoid external dependencies

## Extensibility

### Adding a New Renderer

1. Create package in `pkg/renderer/yourrenderer/`
2. Define a `Source` struct with renderer-specific fields
3. Create `yourrenderer.go` with constructor: `func New(inputs []Source, opts ...RendererOption) (*Renderer, error)`
4. Implement `types.Renderer` interface with `Process(ctx context.Context, values map[string]any)` method
5. Create `yourrenderer_option.go` following the pattern in `pkg/util/option.go`
6. In `Process()`, iterate inputs, render each, apply renderer-specific F/T using `pipeline.ApplyFilters()` and `pipeline.ApplyTransformers()`
7. If renderer supports templates/values, implement deep merge of `values` parameter with Source-level values using `util.DeepMerge()`

**All renderers follow the consistent `[]Source` pattern for type-safe, flexible input handling.**

**Example Renderer Structure:**

```go
// yourrenderer.go
package yourrenderer

import (
    "context"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

type Source struct {
    // Renderer-specific fields
    Path   string
    Values func(context.Context) (map[string]any, error)
}

type Renderer struct {
    inputs []Source
    opts   *RendererOptions
}

func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
    // Validate inputs
    for i, input := range inputs {
        if input.Path == "" {
            return nil, fmt.Errorf("input[%d]: Path is required", i)
        }
    }

    // Initialize options
    rendererOpts := RendererOptions{
        Filters:      make([]types.Filter, 0),
        Transformers: make([]types.Transformer, 0),
    }

    for _, opt := range opts {
        opt.ApplyTo(&rendererOpts)
    }

    return &Renderer{
        inputs: inputs,
        opts:   &rendererOpts,
    }, nil
}

func (r *Renderer) Process(ctx context.Context, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
    allObjects := make([]unstructured.Unstructured, 0)

    for i, input := range r.inputs {
        // Get source values
        sourceValues, err := computeValues(ctx, input, renderTimeValues)
        if err != nil {
            return nil, fmt.Errorf("input[%d]: %w", i, err)
        }

        // Render objects with merged values
        objects, err := r.renderSingle(ctx, input, sourceValues)
        if err != nil {
            return nil, fmt.Errorf("input[%d]: %w", i, err)
        }

        allObjects = append(allObjects, objects...)
    }

    // Apply renderer-specific filters and transformers
    return pipeline.Apply(ctx, allObjects, r.opts.Filters, r.opts.Transformers)
}
```

```go
// yourrenderer_option.go
package yourrenderer

import (
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

type RendererOption = util.Option[RendererOptions]

type RendererOptions struct {
    Filters      []types.Filter
    Transformers []types.Transformer
}

func (opts RendererOptions) ApplyTo(target *RendererOptions) {
    target.Filters = opts.Filters
    target.Transformers = opts.Transformers
}

func WithFilter(f types.Filter) RendererOption {
    return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
        opts.Filters = append(opts.Filters, f)
    })
}

func WithTransformer(t types.Transformer) RendererOption {
    return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
        opts.Transformers = append(opts.Transformers, t)
    })
}
```

### Adding New Filter/Transformer

1. Define a constructor function that returns `types.Filter` or `types.Transformer`
2. If configuration is needed, accept parameters and return a closure
3. Add via `engine.WithFilter`/`engine.WithTransformer` for engine-level or via renderer options

**Example:**

```go
// pkg/filter/custom/custom.go
package custom

import (
    "context"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

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

## Code Review Guidelines

### Linter Rules

All code must pass `make check/lint` before submission. Key linter rules:

- **goconst**: Extract repeated string literals to constants
- **gosec**: No hardcoded secrets (use `//nolint:gosec` only for test data with comment explaining why)
- **staticcheck**: Follow all suggestions (prefer tagged switch over if/else chains, etc.)
- **Comment formatting**: All comments must end with periods

### Git Commit Conventions

**Commit Message Format:**
```
<type>: <subject>

<body>

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring (no functional changes)
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `build`: Build system or dependency changes
- `chore`: Maintenance tasks

**Subject:**
- Use imperative mood ("add feature" not "added feature")
- Don't capitalize first letter
- No period at the end
- Max 72 characters

**Body:**
- Explain what and why (not how)
- Separate from subject with blank line
- Wrap at 72 characters
- Use bullet points for multiple items

**Example:**
```
feat: enhance kustomize renderer with virtual filesystem and configuration options

This commit introduces several improvements to the kustomize renderer:

**Virtual Filesystem (unionfs)**
- Add memory-layer union filesystem for zero-disk-I/O operations
- Support in-memory file overlays over delegate filesystem

**LoadRestrictions Configuration**
- Add two-level LoadRestrictions configuration
- Default to LoadRestrictionsRootOnly for security

Breaking Changes: None - all changes are additive.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Pull Request Checklist

Before submitting a PR:
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make check/lint`)
- [ ] Code formatted (`make fmt`)
- [ ] New tests added for new features
- [ ] Documentation updated (design.md, README.md, or development.md as needed)
- [ ] All test data extracted to package-level constants
- [ ] Benchmark tests follow naming convention
- [ ] Error handling follows conventions
- [ ] Functional options pattern used for configuration

### Code Style

- **Function signatures**: Each parameter must have its own type declaration (never group parameters with same type)
- **Comments**: Explain *why*, not *what*. Focus on non-obvious behavior, edge cases, and relationships
- **Error wrapping**: Always use `fmt.Errorf` with `%w` for error chains
- **Context propagation**: Pass context through all layers for cancellation support
- **Zero values**: Leverage zero value semantics instead of pointers where appropriate
