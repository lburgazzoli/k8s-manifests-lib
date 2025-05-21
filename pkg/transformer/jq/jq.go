package jq

import (
	"context"
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewTransformer creates a new JQ transformer with the given expression and options
func NewTransformer(expression string, opts ...Option) (engine.Transformer, error) {
	t := &options{}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, err
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		v, ok := code.Run(obj.Object).Next()
		if !ok {
			return unstructured.Unstructured{}, fmt.Errorf("jq expression returned no results")
		}

		ret := unstructured.Unstructured{}

		switch v := v.(type) {
		case error:
			return ret, v
		case map[string]interface{}:
			data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(v)
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
