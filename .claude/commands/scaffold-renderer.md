---
description: Scaffold a new renderer with boilerplate code including caching support
---

You are tasked with creating a new renderer implementation for k8s-manifests-lib.

The user will invoke this command with: `/scaffold-renderer <renderer-name>`

Extract the renderer name from the user's message (everything after `/scaffold-renderer`). If no name is provided, ask the user for the renderer name.

Use the renderer name (in lowercase) to:

1. Create the package directory: `pkg/renderer/{name}/`

2. Create `pkg/renderer/{name}/{name}.go` with:
   - Package declaration
   - Required imports (context, fmt, slices, k8s.io/apimachinery/pkg/apis/meta/v1/unstructured, k8s.io/apimachinery/pkg/runtime, k8s.io/apimachinery/pkg/runtime/serializer/yaml)
   - Additional imports for the renderer (github.com/lburgazzoli/k8s-manifests-lib/pkg/types, github.com/lburgazzoli/k8s-manifests-lib/pkg/util, github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache)
   - Source struct with appropriate fields for the renderer type (FS fs.FS, Path string for file-based, or other relevant fields)
   - If the renderer uses dynamic values, add a Values field: `Values func(context.Context) (T, error)` where T is the appropriate type
   - Renderer struct with fields: inputs []Source, filters []types.Filter, transformers []types.Transformer, decoder runtime.Decoder, cache cache.Interface[[]unstructured.Unstructured]
   - New() constructor that validates inputs and accepts RendererOption
   - Process(ctx context.Context) method that implements the rendering pipeline
   - renderSingle(ctx context.Context, input Source) method for processing individual inputs
   - If Values function is used, add a values() helper method to safely get values with nil check
   - Implement caching in renderSingle: check cache before rendering, store after rendering
   - Apply renderer-specific filters and transformers using util.ApplyFilters() and util.ApplyTransformers()
   - If the renderer has a Values function, add a helper function like helm.Values() or gotemplate.Values() for static values

3. Create `pkg/renderer/{name}/{name}_option.go` with:
   - RendererOption type alias
   - RendererOptions struct with Filters, Transformers, and Cache fields
   - ApplyTo method for RendererOptions
   - WithFilter(filter types.Filter) function option
   - WithTransformer(transformer types.Transformer) function option
   - WithCache(opts ...cache.Option) function option that calls cache.NewRenderCache()

4. Create `pkg/renderer/{name}/{name}_test.go` with:
   - Basic tests using vanilla Gomega (import . "github.com/onsi/gomega")
   - TestRenderer with subtests covering:
     - Basic rendering functionality
     - Filter application
     - Transformer application
     - Error cases
   - TestNew with subtests for input validation
   - TestCacheIntegration with subtests:
     - should cache identical renders
     - should miss cache on different values/inputs
     - should work with cache disabled
     - should return clones from cache (verify modifications don't affect cache)
   - If the renderer has a Values helper, add TestValuesHelper
   - Three benchmark tests:
     - Benchmark{Name}RenderWithoutCache
     - Benchmark{Name}RenderWithCache
     - Benchmark{Name}RenderCacheMiss

Key implementation details:

**Caching Pattern:**
```go
func (r *Renderer) renderSingle(ctx context.Context, input Source) ([]unstructured.Unstructured, error) {
    // Get values if applicable
    values, err := r.values(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to get values: %w", err)
    }

    // Compute cache key (use dump.ForHash for values, or path for file-based)
    var cacheKey string
    if r.cache != nil {
        cacheKey = dump.ForHash(values) // or input.Path for file-based
        r.cache.Sync()
        if cached, found := r.cache.Get(cacheKey); found {
            return cached, nil
        }
    }

    // Render logic here
    result := make([]unstructured.Unstructured, 0)
    // ... rendering implementation ...

    // Cache result
    if r.cache != nil {
        r.cache.Set(cacheKey, result)
    }

    return result, nil
}
```

**Values Helper Pattern (if needed):**
```go
// Values returns a Values function that always returns the provided static values.
func Values(values T) func(context.Context) (T, error) {
    return func(_ context.Context) (T, error) {
        return values, nil
    }
}

func (r *Renderer) values(ctx context.Context, input Source) (T, error) {
    if input.Values == nil {
        return defaultValue, nil
    }

    v, err := input.Values(ctx)
    if err != nil {
        return defaultValue, err
    }

    return v, nil
}
```

**Test Pattern:**
```go
func TestRenderer(t *testing.T) {
    g := NewWithT(t)
    ctx := t.Context()

    t.Run("should render basic resources", func(t *testing.T) {
        renderer, err := {name}.New([]Source{{...}})
        g.Expect(err).ShouldNot(HaveOccurred())

        objects, err := renderer.Process(ctx)
        g.Expect(err).ShouldNot(HaveOccurred())
        g.Expect(objects).Should(HaveLen(expectedCount))
    })
}
```

Follow the patterns from existing renderers (helm, kustomize, gotemplate, yaml) for consistency.

After creating all files, provide a summary of what was created and suggest next steps (implementing the actual rendering logic specific to this renderer type).