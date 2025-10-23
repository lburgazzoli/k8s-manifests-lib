package kustomize

import (
	"context"
	"fmt"
	"path/filepath"

	goyaml "gopkg.in/yaml.v3"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

var (
	//nolint:gochecknoglobals
	kustomizationFiles = []string{
		"kustomization.yaml",
		"kustomization.yml",
		"Kustomization",
	}
)

func computeValues(ctx context.Context, input Source, renderTimeValues map[string]any) (map[string]string, error) {
	sourceValues := map[string]any{}

	if input.Values != nil {
		v, err := input.Values(ctx)
		if err != nil {
			return nil, err
		}
		// Convert map[string]string to map[string]any for merging
		for k, v := range v {
			sourceValues[k] = v
		}
	}

	// Deep merge with render-time values taking precedence
	merged := util.DeepMerge(sourceValues, renderTimeValues)

	// Convert back to map[string]string
	result := make(map[string]string, len(merged))
	for k, v := range merged {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result, nil
}

// writeValuesConfigMap writes values as a ConfigMap YAML file using the renderer's filesystem.
// Returns the path to the written file, or empty string if no values.
func writeValuesConfigMap(fs filesys.FileSystem, path string, values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}

	valuesPath := filepath.Join(path, "values.yaml")

	// Check if file exists
	if fs.Exists(valuesPath) {
		return "", fmt.Errorf("values.yaml already exists at %s, refusing to overwrite", valuesPath)
	}

	// Create ConfigMap structure
	configMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]string{
			"name": "values",
		},
		"data": values,
	}

	data, err := goyaml.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values ConfigMap: %w", err)
	}

	if err := fs.WriteFile(valuesPath, data); err != nil {
		return "", fmt.Errorf("failed to write values.yaml: %w", err)
	}

	return valuesPath, nil
}

func readKustomization(fs filesys.FileSystem, path string) (*kustomizetypes.Kustomization, error) {
	var kustFile string

	for _, filename := range kustomizationFiles {
		candidate := filepath.Join(path, filename)
		if fs.Exists(candidate) {
			kustFile = candidate
			break
		}
	}

	if kustFile == "" {
		return nil, fmt.Errorf("no kustomization file found in %s", path)
	}

	content, err := fs.ReadFile(kustFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read kustomization from %s: %w", kustFile, err)
	}

	kust := &kustomizetypes.Kustomization{}
	if err := kust.Unmarshal(content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kustomization from %s: %w", kustFile, err)
	}

	return kust, nil
}
