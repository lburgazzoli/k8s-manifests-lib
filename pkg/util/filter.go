package util

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

func ApplyFilters(ctx context.Context, objects []unstructured.Unstructured, filters []types.Filter) ([]unstructured.Unstructured, error) {
	if len(filters) == 0 {
		return objects, nil
	}

	filtered := make([]unstructured.Unstructured, 0, len(objects))
	for i, obj := range objects {
		matches, err := matchesAllFilters(ctx, obj, filters)
		if err != nil {
			return nil, fmt.Errorf(
				"filter error for object[%d] %s:%s %s (namespace: %s): %w",
				i,
				obj.GroupVersionKind().GroupVersion(),
				obj.GroupVersionKind().Kind,
				obj.GetName(),
				obj.GetNamespace(),
				err,
			)
		}

		if matches {
			filtered = append(filtered, obj)
		}
	}

	return filtered, nil
}

func matchesAllFilters(ctx context.Context, obj unstructured.Unstructured, filters []types.Filter) (bool, error) {
	for i, filter := range filters {
		ok, err := filter(ctx, obj)
		if err != nil {
			return false, fmt.Errorf("filter[%d] failed: %w", i, err)
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
	for i, obj := range objects {
		result, err := applyAllTransformers(ctx, obj, transformers)
		if err != nil {
			return nil, fmt.Errorf(
				"transformer error for object[%d] %s:%s %s (namespace: %s): %w",
				i,
				obj.GroupVersionKind().GroupVersion(),
				obj.GroupVersionKind().Kind,
				obj.GetName(),
				obj.GetNamespace(),
				err,
			)
		}

		transformed = append(transformed, result)
	}

	return transformed, nil
}

func applyAllTransformers(ctx context.Context, obj unstructured.Unstructured, transformers []types.Transformer) (unstructured.Unstructured, error) {
	result := obj
	for i, transformer := range transformers {
		r, err := transformer(ctx, result)
		if err != nil {
			return unstructured.Unstructured{}, fmt.Errorf("transformer[%d] failed: %w", i, err)
		}

		result = r
	}

	return result, nil
}
