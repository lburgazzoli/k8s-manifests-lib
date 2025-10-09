package yaml

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/pipeline"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/k8s"
)

const rendererType = "yaml"

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
	inputs       []Source
	filters      []types.Filter
	transformers []types.Transformer
	cache        cache.Interface[[]unstructured.Unstructured]
}

// New creates a new YAML Renderer with the given inputs and options.
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
			return nil, fmt.Errorf("error rendering YAML[%d] pattern %s: %w", i, input.Path, err)
		}

		allObjects = append(allObjects, objects...)
	}

	transformed, err := pipeline.Apply(ctx, allObjects, r.filters, r.transformers)
	if err != nil {
		return nil, fmt.Errorf("yaml renderer: %w", err)
	}

	return transformed, nil
}

// Name returns the renderer type identifier.
func (r *Renderer) Name() string {
	return rendererType
}

// renderSingle performs the rendering for a single YAML input.
func (r *Renderer) renderSingle(_ context.Context, data Source) ([]unstructured.Unstructured, error) {
	// Use path as cache key
	cacheKey := data.Path

	// Check cache (if enabled)
	if r.cache != nil {
		// ensure objects are evicted
		r.cache.Sync()

		if cached, found := r.cache.Get(cacheKey); found {
			return cached, nil
		}
	}

	result := make([]unstructured.Unstructured, 0)

	// Find all matching files
	matches, err := fs.Glob(data.FS, data.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to match pattern %s: %w", data.Path, err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no files matched pattern: %s", data.Path)
	}

	// Process each matched file
	for _, match := range matches {
		fileObjects, err := r.loadYAMLFile(data.FS, match)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", match, err)
		}

		result = append(result, fileObjects...)
	}

	// Cache result (if enabled)
	if r.cache != nil {
		r.cache.Set(cacheKey, result)
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
		return nil, fmt.Errorf("path is a directory, not a file: %s", path)
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

	return objects, nil
}
