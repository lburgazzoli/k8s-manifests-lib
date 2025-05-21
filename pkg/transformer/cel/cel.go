package cel

import (
	"context"

	"github.com/google/cel-go/cel"
	"github.com/lburgazzoli/k8s-manifests-lib/pkg/engine"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NewTransformer creates a new CEL transformer with the given expression and options
func NewTransformer(expression string, opts ...Option) (engine.Transformer, error) {
	o := &options{
		envOptions: make([]cel.EnvOption, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(o)
	}

	// Create a CEL environment with the unstructured content as root
	env, err := cel.NewEnv(o.envOptions...)
	if err != nil {
		return nil, err
	}

	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, obj unstructured.Unstructured) (unstructured.Unstructured, error) {
		out, _, err := prg.Eval(obj.Object)
		if err != nil {
			return unstructured.Unstructured{}, err
		}

		if !out.Value().(bool) {
			return unstructured.Unstructured{}, nil
		}

		return obj, nil
	}, nil
}
