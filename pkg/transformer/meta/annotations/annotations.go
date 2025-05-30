package annotations

import (
	"context"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Transform(annotationsToApply map[string]string) types.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetAnnotations()
		if values == nil {
			values = make(map[string]string)
		}

		for k, v := range annotationsToApply {
			values[k] = v
		}

		obj.SetAnnotations(values)

		return obj, nil
	}
}
