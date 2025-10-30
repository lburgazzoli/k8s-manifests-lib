package gotemplate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/dump"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
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

	// Parsed templates (lazy-loaded on first Process call)
	templates *template.Template
}

// Validate checks if the Source configuration is valid.
func (s Source) Validate() error {
	if s.FS == nil {
		return errors.New("fs is required")
	}
	if len(strings.TrimSpace(s.Path)) == 0 {
		return errors.New("path cannot be empty or whitespace-only")
	}
	return nil
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
	inputs []Source
	opts   RendererOptions
}

// New creates a new GoTemplate Renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	// Validate inputs at construction time to fail fast on configuration errors.
	// Checks: FS not nil, Path not empty/whitespace.
	for _, input := range inputs {
		if err := input.Validate(); err != nil {
			return nil, err
		}
	}

	rendererOpts := RendererOptions{
		Filters:      make([]types.Filter, 0),
		Transformers: make([]types.Transformer, 0),
	}

	for _, opt := range opts {
		opt.ApplyTo(&rendererOpts)
	}

	r := &Renderer{
		inputs: slices.Clone(inputs),
		opts:   rendererOpts,
	}

	return r, nil
}

// Process executes the rendering logic for all configured inputs.
func (r *Renderer) Process(ctx context.Context, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for i := range r.inputs {
		if r.inputs[i].templates == nil {
			tmpl, err := template.ParseFS(r.inputs[i].FS, r.inputs[i].Path)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to parse templates (path: %s): %w",
					r.inputs[i].Path,
					err,
				)
			}

			r.inputs[i].templates = tmpl.Option("missingkey=error")
		}

		objects, err := r.renderSingle(ctx, r.inputs[i], renderTimeValues)
		if err != nil {
			return nil, fmt.Errorf("error rendering gotemplate[%d] pattern %s: %w", i, r.inputs[i].Path, err)
		}

		// Apply renderer-level filters and transformers per-source for better error context
		transformed, err := pipeline.Apply(ctx, objects, r.opts.Filters, r.opts.Transformers)
		if err != nil {
			return nil, fmt.Errorf(
				"error applying filters/transformers to gotemplate[%d] pattern %s: %w",
				i,
				r.inputs[i].Path,
				err,
			)
		}

		allObjects = append(allObjects, transformed...)
	}

	return allObjects, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

func (r *Renderer) values(ctx context.Context, input Source, renderTimeValues map[string]any) (any, error) {
	sourceValues := map[string]any{}

	if input.Values != nil {
		v, err := input.Values(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get values for template pattern %q: %w", input.Path, err)
		}

		// If source values are a map, convert to map[string]any for merging
		if vMap, ok := v.(map[string]any); ok {
			sourceValues = vMap
		} else {
			// If not a map, return as-is (can't merge with render-time values)
			// Render-time values would be ignored in this case
			return v, nil
		}
	}

	// Deep merge with render-time values taking precedence
	return util.DeepMerge(sourceValues, renderTimeValues), nil
}

// renderSingle performs the rendering for a single template input.
func (r *Renderer) renderSingle(ctx context.Context, input Source, renderTimeValues map[string]any) ([]unstructured.Unstructured, error) {
	// Get values dynamically (includes render-time values)
	values, err := r.values(ctx, input, renderTimeValues)
	if err != nil {
		return nil, fmt.Errorf("failed to get values for pattern %q: %w", input.Path, err)
	}

	// Compute cache key from template path and values
	type cacheKeyData struct {
		Path   string
		Values any
	}

	var cacheKey string

	// Check cache (if enabled)
	if r.opts.Cache != nil {
		cacheKey = dump.ForHash(cacheKeyData{
			Path:   input.Path,
			Values: values,
		})

		// ensure objects are evicted
		r.opts.Cache.Sync()

		if cached, found := r.opts.Cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	result := make([]unstructured.Unstructured, 0)

	// Execute each template
	for _, t := range input.templates.Templates() {
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

		// Add source annotations if enabled
		if r.opts.SourceAnnotations {
			for i := range objs {
				annotations := objs[i].GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}

				annotations[types.AnnotationSourceType] = rendererType
				annotations[types.AnnotationSourcePath] = input.Path
				annotations[types.AnnotationSourceFile] = t.Name()

				objs[i].SetAnnotations(annotations)
			}
		}

		result = append(result, objs...)
	}

	// Cache result (if enabled)
	if r.opts.Cache != nil {
		r.opts.Cache.Set(cacheKey, result)
	}

	return result, nil
}
