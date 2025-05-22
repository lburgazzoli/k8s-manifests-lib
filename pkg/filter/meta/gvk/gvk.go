package gvk

import (
	"context"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// NewFilter creates a new filter function that filters objects based on their GroupVersionKind.
// An object is kept if its GVK matches any of the provided GVKs.
func NewFilter(gvks ...schema.GroupVersionKind) types.Filter {
	m := make(map[schema.GroupVersionKind]struct{})

	for _, gvk := range gvks {
		m[gvk] = struct{}{}
	}

	return func(ctx context.Context, object unstructured.Unstructured) (bool, error) {
		_, ok := m[object.GetObjectKind().GroupVersionKind()]
		return ok, nil
	}
}
