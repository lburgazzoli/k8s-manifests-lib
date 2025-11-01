package util

import "reflect"

// DeepMerge recursively merges overlay into base, with overlay values taking precedence.
// Returns a new map without modifying the inputs.
//
// Merge Semantics:
//   - Maps: Recursively merged. Keys from both maps are preserved.
//     Overlapping keys use the overlay value.
//   - Slices: Completely replaced by overlay (NOT appended or merged).
//   - Other types: Overlay value replaces base value.
//   - Type mismatches: Overlay value wins regardless of types.
//   - Nil values: Treated as empty - overlay nil returns cloned base, base nil returns cloned overlay.
//
// Examples:
//
// Nested map merge:
//
//	base := map[string]any{
//	    "config": map[string]any{
//	        "host": "localhost",
//	        "port": 8080,
//	        "timeout": 30,
//	    },
//	}
//	overlay := map[string]any{
//	    "config": map[string]any{
//	        "port": 9090,  // Override
//	        "retries": 3,  // Add new
//	    },
//	}
//	result := DeepMerge(base, overlay)
//	// result["config"] = {"host": "localhost", "port": 9090, "timeout": 30, "retries": 3}
//
// Slice replacement (NOT merge):
//
//	base := map[string]any{"tags": []string{"dev", "test"}}
//	overlay := map[string]any{"tags": []string{"prod"}}
//	result := DeepMerge(base, overlay)
//	// result["tags"] = ["prod"]  // NOT ["dev", "test", "prod"]
//
// Type mismatch (overlay wins):
//
//	base := map[string]any{"service": map[string]any{"type": "ClusterIP"}}
//	overlay := map[string]any{"service": "NodePort"}
//	result := DeepMerge(base, overlay)
//	// result["service"] = "NodePort"  // Map replaced by string
//
// Use Case - Render-Time Values:
//
//	// Configuration-time values
//	source := helm.Source{
//	    Values: helm.Values(map[string]any{
//	        "replicaCount": 2,
//	        "image": map[string]any{
//	            "repository": "nginx",
//	            "tag": "1.25.0",
//	            "pullPolicy": "IfNotPresent",
//	        },
//	    }),
//	}
//	// Render-time override (merged with source values)
//	objects, err := engine.Render(ctx, engine.WithValues(map[string]any{
//	    "replicaCount": 5,           // Override
//	    "image": map[string]any{
//	        "tag": "1.26.0",          // Override tag only
//	        // repository and pullPolicy preserved from source
//	    },
//	}))
//	// Final values: {replicaCount: 5, image: {repository: "nginx", tag: "1.26.0", pullPolicy: "IfNotPresent"}}
func DeepMerge(base map[string]any, overlay map[string]any) map[string]any {
	if base == nil && overlay == nil {
		return map[string]any{}
	}
	if base == nil {
		return cloneMap(overlay)
	}
	if overlay == nil {
		return cloneMap(base)
	}

	// Preallocate result map with estimated capacity
	result := make(map[string]any, len(base)+len(overlay))

	// First, copy base values that won't be overridden by overlay
	// This avoids cloning values that will be immediately replaced
	for k, baseValue := range base {
		if overlayValue, willOverride := overlay[k]; willOverride {
			// Check if both are maps - if so, we'll merge recursively
			baseMap, baseIsMap := baseValue.(map[string]any)
			overlayMap, overlayIsMap := overlayValue.(map[string]any)

			if baseIsMap && overlayIsMap {
				// Recursively merge nested maps
				result[k] = DeepMerge(baseMap, overlayMap)
			} else {
				// Overlay wins for non-map values or type mismatches
				result[k] = cloneValue(overlayValue)
			}
		} else {
			// Base value not overridden - clone and keep it
			result[k] = cloneValue(baseValue)
		}
	}

	// Add keys that only exist in overlay
	for k, overlayValue := range overlay {
		if _, exists := base[k]; !exists {
			result[k] = cloneValue(overlayValue)
		}
	}

	return result
}

// cloneMap creates a shallow copy of a map.
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = cloneValue(v)
	}

	return result
}

// cloneValue creates a deep copy of a value.
// For maps, recursively clones all nested maps and slices.
// For []any slices, recursively clones all elements with deep cloning.
// For other slice types ([]string, []int, etc.), creates a shallow copy of the slice itself.
// For primitives and other types, returns the value as-is.
func cloneValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		return cloneMap(val)
	case []any:
		clone := make([]any, len(val))
		for i, elem := range val {
			clone[i] = cloneValue(elem)
		}

		return clone
	// Common typed slices - use type switches for performance instead of reflection
	case []string:
		clone := make([]string, len(val))
		copy(clone, val)

		return clone
	case []int:
		clone := make([]int, len(val))
		copy(clone, val)

		return clone
	case []int64:
		clone := make([]int64, len(val))
		copy(clone, val)

		return clone
	case []float64:
		clone := make([]float64, len(val))
		copy(clone, val)

		return clone
	case []bool:
		clone := make([]bool, len(val))
		copy(clone, val)

		return clone
	default:
		// Handle other slice types using reflection to avoid shared memory
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice {
			sliceLen := rv.Len()
			clone := reflect.MakeSlice(rv.Type(), sliceLen, sliceLen)
			for i := range sliceLen {
				clone.Index(i).Set(rv.Index(i))
			}

			return clone.Interface()
		}

		return v
	}
}
