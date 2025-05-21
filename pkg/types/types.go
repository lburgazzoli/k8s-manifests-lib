package types

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Filter is a function type that processes a single unstructured.Unstructured object
// and returns true if the object should be kept, or false if it should be discarded.
type Filter func(ctx context.Context, object unstructured.Unstructured) (bool, error)

// Transformer is a function type that processes a single unstructured.Unstructured object
// and returns the transformed object.
type Transformer func(ctx context.Context, object unstructured.Unstructured) (unstructured.Unstructured, error)

// Renderer is a non-generic interface that concrete renderer types implement.
// This allows the Engine to manage them heterogeneously.
type Renderer interface {
	// Process executes the rendering logic for all configured inputs of this renderer.
	Process(ctx context.Context) ([]unstructured.Unstructured, error)
}
