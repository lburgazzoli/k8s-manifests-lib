package kustomize

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/yaml"
)

// renderSingle performs the rendering for a single Kustomize directory.
func (r *Renderer) renderSingle(ctx context.Context, data Data) ([]unstructured.Unstructured, error) {
	// Create a filesystem for Kustomize
	fs := filesys.MakeFsOnDisk()

	// Create Kustomize options
	opts := krusty.MakeDefaultOptions()
	opts.LoadRestrictions = types.LoadRestrictionsNone

	// Create a Kustomize builder
	k := krusty.MakeKustomizer(opts)

	// Build the resources
	resMap, err := k.Run(fs, data.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to run kustomize build: %w", err)
	}

	// Convert the resources to unstructured objects
	var objects []unstructured.Unstructured
	for _, res := range resMap.Resources() {
		// Convert the resource to YAML
		yamlBytes, err := res.AsYAML()
		if err != nil {
			return nil, fmt.Errorf("failed to convert resource to YAML: %w", err)
		}

		// Parse the YAML into an unstructured object
		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal(yamlBytes, obj); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}

		objects = append(objects, *obj)
	}

	return objects, nil
}
