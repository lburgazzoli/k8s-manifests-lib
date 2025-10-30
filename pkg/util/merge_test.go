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

	t.Run("should deeply clone nested maps in slices", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"items": []any{
				map[string]any{
					"name": "item1",
					"config": map[string]any{
						"enabled": true,
					},
				},
			},
		}
		overlay := map[string]any{
			"items": []any{
				map[string]any{
					"name": "item2",
				},
			},
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(Equal(map[string]any{
			"items": []any{
				map[string]any{
					"name": "item2",
				},
			},
		}))

		resultItems := result["items"].([]any)
		overlayItems := overlay["items"].([]any)
		resultMap := resultItems[0].(map[string]any)
		overlayMap := overlayItems[0].(map[string]any)

		resultMap["modified"] = "value"

		g.Expect(overlayMap).ShouldNot(HaveKey("modified"))
	})

	t.Run("should deeply clone nested slices in slices", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"matrix": []any{
				[]any{"a", "b"},
				[]any{"c", "d"},
			},
		}
		overlay := map[string]any{
			"other": "value",
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result).Should(HaveKey("matrix"))
		g.Expect(result).Should(HaveKey("other"))

		resultMatrix := result["matrix"].([]any)
		baseMatrix := base["matrix"].([]any)

		resultInner := resultMatrix[0].([]any)
		baseInner := baseMatrix[0].([]any)

		resultInner[0] = "modified"

		g.Expect(baseInner[0]).Should(Equal("a"))
	})

	t.Run("should deeply clone complex nested structures with mixed types", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "nginx",
					"image": "nginx:1.0",
					"env": []any{
						map[string]any{
							"name":  "ENV_VAR",
							"value": "base_value",
						},
					},
				},
			},
		}
		overlay := map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "nginx",
					"image": "nginx:2.0",
					"env": []any{
						map[string]any{
							"name":  "ENV_VAR",
							"value": "overlay_value",
						},
					},
				},
			},
		}

		result := util.DeepMerge(base, overlay)

		resultContainers := result["containers"].([]any)
		overlayContainers := overlay["containers"].([]any)
		baseContainers := base["containers"].([]any)

		resultContainer := resultContainers[0].(map[string]any)
		resultEnv := resultContainer["env"].([]any)
		resultEnvVar := resultEnv[0].(map[string]any)

		resultEnvVar["modified"] = "test"

		overlayContainer := overlayContainers[0].(map[string]any)
		overlayEnv := overlayContainer["env"].([]any)
		overlayEnvVar := overlayEnv[0].(map[string]any)
		g.Expect(overlayEnvVar).ShouldNot(HaveKey("modified"))

		baseContainer := baseContainers[0].(map[string]any)
		baseEnv := baseContainer["env"].([]any)
		baseEnvVar := baseEnv[0].(map[string]any)
		g.Expect(baseEnvVar).ShouldNot(HaveKey("modified"))
	})

	t.Run("should clone slices with nil elements", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"list": []any{"a", nil, "c"},
		}
		overlay := map[string]any{
			"other": "value",
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result["list"]).Should(Equal([]any{"a", nil, "c"}))

		resultList := result["list"].([]any)
		baseList := base["list"].([]any)

		resultList[0] = "modified"

		g.Expect(baseList[0]).Should(Equal("a"))
	})

	t.Run("should clone empty slices correctly", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"empty": []any{},
		}
		overlay := map[string]any{
			"other": "value",
		}

		result := util.DeepMerge(base, overlay)

		g.Expect(result["empty"]).Should(Equal([]any{}))
		g.Expect(result["empty"]).ShouldNot(BeIdenticalTo(base["empty"]))
	})

	t.Run("should handle slice of maps with varying structures", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"configs": []any{
				map[string]any{
					"type":  "database",
					"host":  "localhost",
					"ports": []any{5432, 5433},
				},
				map[string]any{
					"type": "cache",
					"ttl":  300,
				},
			},
		}
		overlay := map[string]any{
			"version": "v2",
		}

		result := util.DeepMerge(base, overlay)

		resultConfigs := result["configs"].([]any)
		baseConfigs := base["configs"].([]any)

		resultFirst := resultConfigs[0].(map[string]any)
		resultPorts := resultFirst["ports"].([]any)
		resultPorts[0] = 9999

		baseFirst := baseConfigs[0].(map[string]any)
		basePorts := baseFirst["ports"].([]any)
		g.Expect(basePorts[0]).Should(Equal(5432))
	})

	t.Run("should clone []string slices to avoid shared memory", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"tags": []string{"dev", "test"},
		}

		result := util.DeepMerge(base, nil)

		// Modify the cloned slice
		resultTags := result["tags"].([]string)
		resultTags[0] = "production"

		// Original should be unchanged
		baseTags := base["tags"].([]string)
		g.Expect(baseTags[0]).Should(Equal("dev"))
		g.Expect(baseTags[1]).Should(Equal("test"))
	})

	t.Run("should clone []int slices to avoid shared memory", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"ports": []int{8080, 9090},
		}

		result := util.DeepMerge(base, nil)

		// Modify the cloned slice
		resultPorts := result["ports"].([]int)
		resultPorts[0] = 3000

		// Original should be unchanged
		basePorts := base["ports"].([]int)
		g.Expect(basePorts[0]).Should(Equal(8080))
		g.Expect(basePorts[1]).Should(Equal(9090))
	})

	t.Run("should clone []bool slices to avoid shared memory", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"features": []bool{true, false, true},
		}

		result := util.DeepMerge(base, nil)

		// Modify the cloned slice
		resultFeatures := result["features"].([]bool)
		resultFeatures[0] = false

		// Original should be unchanged
		baseFeatures := base["features"].([]bool)
		g.Expect(baseFeatures[0]).Should(BeTrue())
		g.Expect(baseFeatures[1]).Should(BeFalse())
		g.Expect(baseFeatures[2]).Should(BeTrue())
	})

	t.Run("should handle nested maps with typed slices", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"config": map[string]any{
				"hosts": []string{"localhost", "127.0.0.1"},
				"ports": []int{8080, 9090},
			},
		}

		overlay := map[string]any{
			"config": map[string]any{
				"enabled": true,
			},
		}

		result := util.DeepMerge(base, overlay)

		// Modify the cloned slices
		resultConfig := result["config"].(map[string]any)
		resultHosts := resultConfig["hosts"].([]string)
		resultPorts := resultConfig["ports"].([]int)
		resultHosts[0] = "example.com"
		resultPorts[0] = 3000

		// Original should be unchanged
		baseConfig := base["config"].(map[string]any)
		baseHosts := baseConfig["hosts"].([]string)
		basePorts := baseConfig["ports"].([]int)
		g.Expect(baseHosts[0]).Should(Equal("localhost"))
		g.Expect(basePorts[0]).Should(Equal(8080))
	})

	t.Run("should clone []string in overlay without affecting original", func(t *testing.T) {
		g := NewWithT(t)

		overlay := map[string]any{
			"environments": []string{"dev", "staging", "prod"},
		}

		result := util.DeepMerge(nil, overlay)

		// Modify the cloned slice
		resultEnvs := result["environments"].([]string)
		resultEnvs[2] = "production"

		// Original overlay should be unchanged
		overlayEnvs := overlay["environments"].([]string)
		g.Expect(overlayEnvs[0]).Should(Equal("dev"))
		g.Expect(overlayEnvs[1]).Should(Equal("staging"))
		g.Expect(overlayEnvs[2]).Should(Equal("prod"))
	})
}
