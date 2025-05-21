package labels

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
)

func NewTransformer(labelsToApply map[string]string) engine.Transformer {
	return func(_ context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		values := obj.GetLabels()
		if values == nil {
			values = make(map[string]string)
		}

		for k, v := range labelsToApply {
			values[k] = v
		}

		obj.SetLabels(values)

		return obj, nil
	}
}
