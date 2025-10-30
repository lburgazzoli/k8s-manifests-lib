package util

import "reflect"

// DeepMerge recursively merges overlay into base, with overlay values taking precedence.
// For maps, the merge is recursive. For all other types, overlay replaces base.
// Returns a new map without modifying the inputs.
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

	result := cloneMap(base)

	for k, overlayValue := range overlay {
		baseValue, exists := result[k]

		if !exists {
			result[k] = cloneValue(overlayValue)
			continue
		}

		// Both base and overlay have this key
		baseMap, baseIsMap := baseValue.(map[string]any)
		overlayMap, overlayIsMap := overlayValue.(map[string]any)

		if baseIsMap && overlayIsMap {
			// Recursively merge nested maps
			result[k] = DeepMerge(baseMap, overlayMap)
		} else {
			// Overlay wins for non-map values or type mismatches
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
