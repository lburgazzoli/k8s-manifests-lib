package jq

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/types"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/jq"
)

// Filter creates a new JQ filter with the given expression and options.
func Filter(expression string, opts ...jq.EngineOption) (types.Filter, error) {
	// Create a new JQ engine
	engine, err := jq.NewEngine(expression, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating jq engine: %w", err)
	}

	return func(ctx context.Context, obj unstructured.Unstructured) (bool, error) {
		// Run the JQ program and get a single value
		v, err := engine.Run(obj.Object)
		if err != nil {
			return false, fmt.Errorf("error executing jq expression: %w", err)
		}

		// Convert the result to a boolean
		if b, ok := v.(bool); ok {
			return b, nil
		}

		return false, fmt.Errorf("jq expression must return a boolean, got %T", v)
	}, nil
}
