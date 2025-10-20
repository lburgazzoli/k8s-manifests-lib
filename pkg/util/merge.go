package util

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

// cloneValue creates a copy of a value.
// For maps, creates a shallow copy. For slices, creates a shallow copy.
// For other types, returns the value as-is (primitives, pointers, etc.).
func cloneValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]any:
		return cloneMap(val)
	case []any:
		clone := make([]any, len(val))
		copy(clone, val)
		return clone
	default:
		return v
	}
}
