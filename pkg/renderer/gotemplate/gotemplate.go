package gotemplate

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"slices"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/dump"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/k8s"
)

const rendererType = "gotemplate"

// Source represents the input for a GoTemplate rendering operation.
type Source struct {
	// FS is the filesystem containing template files.
	// Supports embedded filesystems via embed.FS or testing via fstest.MapFS.
	FS fs.FS

	// Path specifies the glob pattern to match template files.
	// Examples: "templates/*.tpl", "**/*.yaml.gotmpl"
	Path string

	// Values provides data to be substituted into templates during rendering.
	// Function is called during rendering to obtain dynamic values.
	// Accessible within templates via dot notation (e.g., {{ .FieldName }}).
	Values func(context.Context) (any, error)
}

// Values returns a Values function that always returns the provided static values.
// This is a convenience helper for the common case of non-dynamic values.
func Values(values any) func(context.Context) (any, error) {
	return func(_ context.Context) (any, error) {
		return values, nil
	}
}

// Renderer handles Go template rendering operations.
// It implements types.Renderer.
type Renderer struct {
	inputs       []Source
	filters      []types.Filter
	transformers []types.Transformer
	cache        cache.Interface[[]unstructured.Unstructured]
}

// New creates a new GoTemplate Renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	// Validate inputs
	for i, input := range inputs {
		if input.FS == nil {
			return nil, fmt.Errorf("input[%d]: FS is required", i)
		}
		if input.Path == "" {
			return nil, fmt.Errorf("input[%d]: Path is required", i)
		}
	}

	r := &Renderer{
		inputs:       slices.Clone(inputs),
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
	}
	for _, opt := range opts {
		opt.ApplyTo(r)
	}

	return r, nil
}

// Process executes the rendering logic for all configured inputs.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for i, input := range r.inputs {
		objects, err := r.renderSingle(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error rendering gotemplate[%d] pattern %s: %w", i, input.Path, err)
		}

		allObjects = append(allObjects, objects...)
	}

	transformed, err := pipeline.Apply(ctx, allObjects, r.filters, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("gotemplate renderer: %w", err)
	}

	return transformed, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

func (r *Renderer) values(ctx context.Context, input Source) (any, error) {
	if input.Values == nil {
		return map[string]any{}, nil
	}

	v, err := input.Values(ctx)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// renderSingle performs the rendering for a single template input.
func (r *Renderer) renderSingle(ctx context.Context, input Source) ([]unstructured.Unstructured, error) {
	// Get values dynamically
	values, err := r.values(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get values: %w", err)
	}

	// Compute cache key from values
	var cacheKey string

	// Check cache (if enabled)
	if r.cache != nil {
		cacheKey = dump.ForHash(values)

		// ensure objects are evicted
		r.cache.Sync()

		if cached, found := r.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	// Create a template with all files matching the path pattern
	tmpl, err := template.ParseFS(input.FS, input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	tmpl = tmpl.Option("missingkey=error")

	result := make([]unstructured.Unstructured, 0)

	// Execute each template
	for _, t := range tmpl.Templates() {
		// Skip the root template
		if t.Name() == "" {
			continue
		}

		// Execute the template
		var buf bytes.Buffer
		if err := t.Execute(&buf, values); err != nil {
			return nil, fmt.Errorf("failed to execute template %s: %w", t.Name(), err)
		}

		// Decode the rendered output into unstructured objects
		objs, err := k8s.DecodeYAML(buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML from template %s: %w", t.Name(), err)
		}

		result = append(result, objs...)
	}

	// Cache result (if enabled)
	if r.cache != nil {
		r.cache.Set(cacheKey, result)
	}

	return result, nil
}
