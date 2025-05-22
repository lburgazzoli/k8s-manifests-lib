package jq

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/jq"
)

// NewTransformer creates a new JQ transformer with the given expression and options
func NewTransformer(expression string, opts ...jq.Option) (types.Transformer, error) {
	// Create a new JQ engine
	engine, err := jq.NewEngine(expression, opts...)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		v, err := engine.Run(obj.Object)
		if err != nil {
			return unstructured.Unstructured{}, err
		}

		ret := unstructured.Unstructured{}

		switch v := v.(type) {
		case map[string]any:
			data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&v)
			if err != nil {
				return ret, fmt.Errorf("failed to convert jq result to unstructured: %w", err)
			}

			ret.SetUnstructuredContent(data)
			return ret, nil
		default:
			return ret, fmt.Errorf("jq expression must return an object, got %T", v)
		}
	}, nil
}
