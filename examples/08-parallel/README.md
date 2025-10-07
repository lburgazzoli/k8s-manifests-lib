# Parallel Rendering Example

This example demonstrates how to enable parallel rendering of manifests, which can significantly improve performance when processing multiple I/O-bound renderers (Helm charts, Kustomize directories, YAML files).

## Features

- **Sequential Rendering**: Default behavior where renderers execute one after another
- **Parallel Rendering**: Concurrent execution of all renderers using goroutines
- **Struct-based Options**: Alternative syntax using struct-based configuration

## When to Use Parallel Rendering

Parallel rendering provides performance benefits when:

1. **Multiple I/O-bound Renderers**: Helm chart fetches from OCI registries, Kustomize builds, or YAML file reads
2. **Independent Renderers**: Renderers that don't depend on each other's output
3. **Operator/Controller Context**: Long-running processes that benefit from faster manifest processing

## Performance Characteristics

- **Best Case**: N renderers with similar execution time → ~N times faster
- **Memory**: Slightly higher due to goroutine overhead (~10 extra allocations)
- **Trade-off**: For very fast in-memory renderers, goroutine overhead may exceed benefits

## Usage

### Function-based Options

```go
engine := engine.New(
    engine.WithRenderer(helmRenderer),
    engine.WithRenderer(kustomizeRenderer),
    engine.WithRenderer(yamlRenderer),
    engine.WithParallel(true), // Enable parallel rendering
)
```

### Struct-based Options

```go
engine := engine.New(&engine.EngineOptions{
    Renderers: []types.Renderer{
        helmRenderer,
        kustomizeRenderer,
        yamlRenderer,
    },
    Parallel: true,
})
```

## Running the Example

```bash
cd examples/08-parallel
go run main.go
```

## Expected Output

```
=== Sequential Rendering ===
Rendered 3 objects in 500µs

=== Parallel Rendering ===
Rendered 3 objects in 200µs

=== Struct-based Options ===
Rendered 3 objects in 200µs

Speedup: 2.50x faster
```

## Implementation Details

- Spawns one goroutine per renderer (no artificial limits)
- Uses channels to collect results
- Preserves error reporting with renderer index
- Thread-safe for concurrent execution
- Default: `parallel=false` (sequential) for backward compatibility
