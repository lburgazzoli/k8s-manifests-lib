package gotemplate

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"text/template"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

// Data represents the input for a GoTemplate rendering operation.
type Data struct {
	FS     fs.FS  // Filesystem containing the templates
	Path   string // Path pattern to match templates (can include globs)
	Values any    // Values to apply to the templates
}

// Renderer handles Go template rendering operations.
// It implements types.Renderer.
type Renderer struct {
	inputs       []Data
	filters      []types.Filter
	transformers []types.Transformer
	decoder      runtime.Decoder
}

// New creates a new GoTemplate Renderer with the given inputs and options.
func New(inputs []Data, opts ...Option) *Renderer {
	r := &Renderer{
		inputs:       inputs,
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
		decoder:      yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Process executes the rendering logic for all configured inputs.
func (r *Renderer) Process(ctx context.Context) ([]unstructured.Unstructured, error) {
	var allObjects []unstructured.Unstructured

	for _, input := range r.inputs {
		objects, err := r.renderSingle(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error rendering template %s: %w", input.Path, err)
		}
		allObjects = append(allObjects, objects...)
	}

	// Apply filters
	filtered, err := util.ApplyFilters(ctx, allObjects, r.filters)
	if err != nil {
		return nil, fmt.Errorf("error applying filters: %w", err)
	}

	// Apply transformers
	transformed, err := util.ApplyTransformers(ctx, filtered, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("error applying transformers: %w", err)
	}

	return transformed, nil
}

// renderSingle performs the rendering for a single template input.
func (r *Renderer) renderSingle(_ context.Context, data Data) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	// Create a template with all files matching the path pattern
	tmpl, err := template.ParseFS(data.FS, data.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	tmpl = tmpl.Option("missingkey=error")

	// Execute each template
	for _, t := range tmpl.Templates() {
		// Skip the root template
		if t.Name() == "" {
			continue
		}

		// Execute the template
		var buf bytes.Buffer
		if err := t.Execute(&buf, data.Values); err != nil {
			return nil, fmt.Errorf("failed to execute template %s: %w", t.Name(), err)
		}

		// Decode the rendered output into unstructured objects
		objs, err := util.DecodeYAML(r.decoder, buf.Bytes())
		if err != nil {
			return nil, fmt.Errorf("failed to decode YAML from template %s: %w", t.Name(), err)
		}

		objects = append(objects, objs...)
	}

	return objects, nil
}
