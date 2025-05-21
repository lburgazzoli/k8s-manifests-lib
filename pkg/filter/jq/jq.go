package jq

import (
	"context"

	"github.com/itchyny/gojq"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func NewFilter(expression string) (types.Filter, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, err
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, obj unstructured.Unstructured) (bool, error) {
		iter := code.Run(obj.Object)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return false, err
			}
			if b, ok := v.(bool); ok && b {
				return true, nil
			}
		}
		return false, nil
	}, nil
}
