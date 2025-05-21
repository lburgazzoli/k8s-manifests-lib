package util

import (
	"context"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ApplyFilters(ctx context.Context, objects []unstructured.Unstructured, filters []types.Filter) ([]unstructured.Unstructured, error) {
	if len(filters) == 0 {
		return objects, nil
	}

	filtered := make([]unstructured.Unstructured, 0, len(objects))
	for _, obj := range objects {
		matches, err := matchesAllFilters(ctx, obj, filters)
		if err != nil {
			return nil, err
		}
		if matches {
			filtered = append(filtered, obj)
		}
	}

	return filtered, nil
}

func matchesAllFilters(ctx context.Context, obj unstructured.Unstructured, filters []types.Filter) (bool, error) {
	for _, filter := range filters {
		ok, err := filter(ctx, obj)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func ApplyTransformers(ctx context.Context, objects []unstructured.Unstructured, transformers []types.Transformer) ([]unstructured.Unstructured, error) {
	if len(transformers) == 0 {
		return objects, nil
	}

	transformed := make([]unstructured.Unstructured, 0, len(objects))
	for _, obj := range objects {
		result, err := applyAllTransformers(ctx, obj, transformers)
		if err != nil {
			return nil, err
		}
		transformed = append(transformed, result)
	}

	return transformed, nil
}

func applyAllTransformers(ctx context.Context, obj unstructured.Unstructured, transformers []types.Transformer) (unstructured.Unstructured, error) {
	result := obj
	for _, transformer := range transformers {
		r, err := transformer(ctx, result)
		if err != nil {
			return unstructured.Unstructured{}, err
		}

		result = r
	}
	return result, nil
}
