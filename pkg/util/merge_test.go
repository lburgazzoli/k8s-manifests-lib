package util_test

import (
	"testing"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"

	. "github.com/onsi/gomega"
)

func TestDeepMerge(t *testing.T) {
	t.Run("should return empty map when both inputs are nil", func(t *testing.T) {
		g := NewWithT(t)

		result := util.DeepMerge(nil, nil)

		g.Expect(result).Should(BeEmpty())
	})

	t.Run("should return clone of overlay when base is nil", func(t *testing.T) {
		g := NewWithT(t)

		overlay := map[string]any{
			"key": "value",
		}

		result := util.DeepMerge(nil, overlay)

		g.Expect(result).Should(Equal(overlay))
		g.Expect(result).ShouldNot(BeIdenticalTo(overlay))
	})

	t.Run("should return clone of base when overlay is nil", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"key": "value",
		}

		result := util.DeepMerge(base, nil)

		g.Expect(result).Should(Equal(base))
		g.Expect(result).ShouldNot(BeIdenticalTo(base))
	})

	t.Run("should merge non-overlapping keys", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"key1": "value1",
		}
		overlay := map[string]any{
			"key2": "value2",
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"key1": "value1",
			"key2": "value2",
		}))
	})

	t.Run("should override base values with overlay values", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"key": "base_value",
		}
		overlay := map[string]any{
			"key": "overlay_value",
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"key": "overlay_value",
		}))
	})

	t.Run("should deep merge nested maps", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"nested": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		}
		overlay := map[string]any{
			"nested": map[string]any{
				"key2": "new_value2",
				"key3": "value3",
			},
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"nested": map[string]any{
				"key1": "value1",
				"key2": "new_value2",
				"key3": "value3",
			},
		}))
	})

	t.Run("should override nested map with non-map value", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"key": map[string]any{
				"nested": "value",
			},
		}
		overlay := map[string]any{
			"key": "string_value",
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"key": "string_value",
		}))
	})

	t.Run("should handle complex nested structures", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"replicaCount": 1,
			"image": map[string]any{
				"repository": "nginx",
				"tag":        "v1.0",
				"pullPolicy": "IfNotPresent",
			},
			"service": map[string]any{
				"type": "ClusterIP",
				"port": 80,
			},
		}
		overlay := map[string]any{
			"replicaCount": 3,
			"image": map[string]any{
				"tag": "v2.0",
			},
			"resources": map[string]any{
				"limits": map[string]any{
					"cpu": "100m",
				},
			},
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"replicaCount": 3,
			"image": map[string]any{
				"repository": "nginx",
				"tag":        "v2.0",
				"pullPolicy": "IfNotPresent",
			},
			"service": map[string]any{
				"type": "ClusterIP",
				"port": 80,
			},
			"resources": map[string]any{
				"limits": map[string]any{
					"cpu": "100m",
				},
			},
		}))
	})

	t.Run("should not modify input maps", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"key": "base_value",
		}
		overlay := map[string]any{
			"key": "overlay_value",
		}

		baseOriginal := map[string]any{
			"key": "base_value",
		}
		overlayOriginal := map[string]any{
			"key": "overlay_value",
		}

		_ = util.DeepMerge(base, overlay)

		g.Expect(base).Should(Equal(baseOriginal))
		g.Expect(overlay).Should(Equal(overlayOriginal))
	})

	t.Run("should handle slices by cloning", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"list": []any{"a", "b"},
		}
		overlay := map[string]any{
			"list": []any{"c", "d"},
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"list": []any{"c", "d"},
		}))
	})

	t.Run("should handle deeply nested structures", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"key": "base_value",
					},
				},
			},
		}
		overlay := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"key":     "overlay_value",
						"new_key": "new_value",
					},
				},
			},
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"key":     "overlay_value",
						"new_key": "new_value",
					},
				},
			},
		}))
	})
}
