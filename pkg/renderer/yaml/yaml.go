package yaml

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/k8s"
)

const rendererType = "yaml"

var (
	// ErrNoFilesMatched is returned when no files match the specified pattern.
	ErrNoFilesMatched = errors.New("no files matched pattern")

	// ErrPathIsDirectory is returned when a path is a directory instead of a file.
	ErrPathIsDirectory = errors.New("path is a directory, not a file")
)

// Source represents the input for a YAML rendering operation.
type Source struct {
	// FS is the filesystem containing YAML manifest files.
	// Supports embedded filesystems via embed.FS or testing via fstest.MapFS.
	FS fs.FS

	// Path specifies the glob pattern to match YAML files.
	// Only .yaml and .yml files are processed. Examples: "manifests/*.yaml", "**/*.yml"
	Path string
}

// Renderer handles YAML file rendering operations.
// It implements types.Renderer.
type Renderer struct {
	inputs []*sourceHolder
	opts   RendererOptions
}

// New creates a new YAML Renderer with the given inputs and options.
func New(inputs []Source, opts ...RendererOption) (*Renderer, error) {
	rendererOpts := RendererOptions{
		Filters:      make([]types.Filter, 0),
		Transformers: make([]types.Transformer, 0),
	}

	for _, opt := range opts {
		opt.ApplyTo(&rendererOpts)
	}

	// Wrap sources in holders and validate
	holders := make([]*sourceHolder, len(inputs))
	for i := range inputs {
		holders[i] = &sourceHolder{
			Source: inputs[i],
		}
		if err := holders[i].Validate(); err != nil {
			return nil, err
		}
	}

	r := &Renderer{
		inputs: holders,
		opts:   rendererOpts,
	}

	return r, nil
}

// Process executes the rendering logic for all configured inputs.
// Render-time values are ignored by the YAML renderer as it does not support templates.
func (r *Renderer) Process(ctx context.Context, _ map[string]any) ([]unstructured.Unstructured, error) {
	allObjects := make([]unstructured.Unstructured, 0)

	for _, holder := range r.inputs {
		objects, err := r.renderSingle(ctx, holder)
		if err != nil {
			return nil, fmt.Errorf("error rendering YAML pattern %s: %w", holder.Path, err)
		}

		// Apply renderer-level filters and transformers per-source for better error context
		transformed, err := pipeline.Apply(ctx, objects, r.opts.Filters, r.opts.Transformers)
		if err != nil {
			return nil, fmt.Errorf(
				"error applying filters/transformers to YAML pattern %s: %w",
				holder.Path,
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

// renderSingle performs the rendering for a single YAML input.
func (r *Renderer) renderSingle(_ context.Context, holder *sourceHolder) ([]unstructured.Unstructured, error) {
	// Use path as cache key
	cacheKey := holder.Path

	// Check cache (if enabled)
	if r.opts.Cache != nil {
		// ensure objects are evicted
		r.opts.Cache.Sync()

		if cached, found := r.opts.Cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	result := make([]unstructured.Unstructured, 0)

	// Find all matching files
	matches, err := fs.Glob(holder.FS, holder.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to match pattern %s: %w", holder.Path, err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoFilesMatched, holder.Path)
	}

	// Process each matched file
	for _, match := range matches {
		fileObjects, err := r.loadYAMLFile(holder.FS, match)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", match, err)
		}

		result = append(result, fileObjects...)
	}

	// Cache result (if enabled)
	if r.opts.Cache != nil {
		r.opts.Cache.Set(cacheKey, result)
	}

	return result, nil
}

// loadYAMLFile loads and parses a single YAML file.
func (r *Renderer) loadYAMLFile(fsys fs.FS, path string) ([]unstructured.Unstructured, error) {
	// Check if path is a directory
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrPathIsDirectory, path)
	}

	// Skip non-YAML files
	ext := filepath.Ext(path)
	if ext != ".yaml" && ext != ".yml" {
		return nil, nil
	}

	// Read file
	file, err := fsys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decode YAML content
	objects, err := k8s.DecodeYAML(content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	// Add source annotations if enabled
	if r.opts.SourceAnnotations {
		for i := range objects {
			annotations := objects[i].GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}

			annotations[types.AnnotationSourceType] = rendererType
			annotations[types.AnnotationSourceFile] = path

			objects[i].SetAnnotations(annotations)
		}
	}

	return objects, nil
}
