package gotemplate

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"strings"
	"text/template"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Data represents the input for a GoTemplate rendering operation.
type Data struct {
	FS       fs.FS       // Filesystem containing the templates
	BasePath string      // Base path within the FS to start recursive traversal
	Values   interface{} // Values to apply to the templates
}

// Renderer handles Go template rendering operations.
// It implements types.Renderer.
type Renderer struct {
	inputs       []Data
	filters      []types.Filter
	transformers []types.Transformer
}

// New creates a new GoTemplate Renderer with the given inputs and options.
func New(inputs []Data, opts ...Option) *Renderer {
	r := &Renderer{
		inputs:       inputs,
		filters:      make([]types.Filter, 0),
		transformers: make([]types.Transformer, 0),
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
			return nil, fmt.Errorf("error rendering template %s: %w", input.BasePath, err)
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
func (r *Renderer) renderSingle(ctx context.Context, data Data) ([]unstructured.Unstructured, error) {
	var objects []unstructured.Unstructured

	// Walk the filesystem starting from the base path
	err := fs.WalkDir(data.FS, data.BasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-template files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".tpl") {
			return nil
		}

		// Read the template file
		content, err := fs.ReadFile(data.FS, path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		// Create a new template with the file name as the template name
		tmpl, err := template.New(d.Name()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		// Execute the template
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data.Values); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		// Split the rendered output into individual YAML documents
		docs := bytes.Split(buf.Bytes(), []byte("---"))
		for _, doc := range docs {
			// Skip empty documents
			if len(bytes.TrimSpace(doc)) == 0 {
				continue
			}

			// Parse the YAML into an unstructured object
			obj := &unstructured.Unstructured{}
			if err := yaml.Unmarshal(doc, obj); err != nil {
				return fmt.Errorf("failed to parse YAML from template %s: %w", path, err)
			}

			objects = append(objects, *obj)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return objects, nil
}
