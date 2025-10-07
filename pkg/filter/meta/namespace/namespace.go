package namespace

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
)

// Filter returns a filter that keeps objects in the specified namespaces.
// Empty namespace matches cluster-scoped resources.
func Filter(namespaces ...string) types.Filter {
	allowed := make(map[string]bool, len(namespaces))
	for _, ns := range namespaces {
		allowed[ns] = true
	}

	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return allowed[obj.GetNamespace()], nil
	}
}

// Exclude returns a filter that excludes objects from the specified namespaces.
func Exclude(namespaces ...string) types.Filter {
	excluded := make(map[string]bool, len(namespaces))
	for _, ns := range namespaces {
		excluded[ns] = true
	}

	return func(_ context.Context, obj unstructured.Unstructured) (bool, error) {
		return !excluded[obj.GetNamespace()], nil
	}
}
