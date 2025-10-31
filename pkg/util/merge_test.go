package util_test

import (
	"fmt"
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

	t.Run("should clone []uint8 slices via reflection", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"bytes": []uint8{1, 2, 3, 4},
		}

		result := util.DeepMerge(base, nil)

		resultBytes := result["bytes"].([]uint8)
		resultBytes[0] = 255

		baseBytes := base["bytes"].([]uint8)
		g.Expect(baseBytes[0]).Should(Equal(uint8(1)))
		g.Expect(baseBytes).Should(Equal([]uint8{1, 2, 3, 4}))
	})

	t.Run("should clone []float32 slices via reflection", func(t *testing.T) {
		g := NewWithT(t)

		base := map[string]any{
			"values": []float32{1.1, 2.2, 3.3},
		}

		result := util.DeepMerge(base, nil)

		resultValues := result["values"].([]float32)
		resultValues[0] = 9.9

		baseValues := base["values"].([]float32)
		g.Expect(baseValues[0]).Should(Equal(float32(1.1)))
		g.Expect(baseValues).Should(Equal([]float32{1.1, 2.2, 3.3}))
	})

	t.Run("should clone custom struct slices via reflection", func(t *testing.T) {
		g := NewWithT(t)

		type customStruct struct {
			Name  string
			Value int
		}

		base := map[string]any{
			"items": []customStruct{
				{Name: "item1", Value: 100},
				{Name: "item2", Value: 200},
			},
		}

		result := util.DeepMerge(base, nil)

		resultItems := result["items"].([]customStruct)
		resultItems[0].Value = 999

		baseItems := base["items"].([]customStruct)
		g.Expect(baseItems[0].Value).Should(Equal(100))
		g.Expect(baseItems[1].Value).Should(Equal(200))
	})
}

// ExampleDeepMerge_nestedMaps demonstrates how nested maps are recursively merged.
func ExampleDeepMerge_nestedMaps() {
	base := map[string]any{
		"config": map[string]any{
			"host":    "localhost",
			"port":    8080,
			"timeout": 30,
		},
	}

	overlay := map[string]any{
		"config": map[string]any{
			"port":    9090, // Override existing key
			"retries": 3,    // Add new key
		},
	}

	result := util.DeepMerge(base, overlay)

	// The nested "config" map is merged, not replaced
	config := result["config"].(map[string]any)
	fmt.Println("host:", config["host"].(string))    // "localhost" (from base)
	fmt.Println("port:", config["port"].(int))       // 9090 (overridden by overlay)
	fmt.Println("timeout:", config["timeout"].(int)) // 30 (from base)
	fmt.Println("retries:", config["retries"].(int)) // 3 (added by overlay)

	// Output:
	// host: localhost
	// port: 9090
	// timeout: 30
	// retries: 3
}

// ExampleDeepMerge_sliceReplacement demonstrates that slices are replaced, not merged.
func ExampleDeepMerge_sliceReplacement() {
	base := map[string]any{
		"tags": []string{"dev", "test"},
	}

	overlay := map[string]any{
		"tags": []string{"prod"},
	}

	result := util.DeepMerge(base, overlay)

	// Slices are completely replaced, NOT appended or merged
	tags := result["tags"].([]string)
	fmt.Println("tags length:", len(tags)) // 1 (not 3)
	fmt.Println("tags[0]:", tags[0])       // "prod"

	// Output:
	// tags length: 1
	// tags[0]: prod
}

// ExampleDeepMerge_typeMismatch demonstrates that overlay wins when types don't match.
func ExampleDeepMerge_typeMismatch() {
	base := map[string]any{
		"service": map[string]any{
			"type": "ClusterIP",
			"port": 80,
		},
	}

	overlay := map[string]any{
		"service": "NodePort", // String replacing a map
	}

	result := util.DeepMerge(base, overlay)

	// When types mismatch, overlay value completely replaces base value
	service := result["service"].(string)
	fmt.Println("service:", service) // "NodePort"

	// Output:
	// service: NodePort
}

// ExampleDeepMerge_renderTimeValues demonstrates a practical use case for render-time value overrides.
func ExampleDeepMerge_renderTimeValues() {
	// Configuration-time values (from Source.Values)
	configValues := map[string]any{
		"replicaCount": 2,
		"image": map[string]any{
			"repository": "nginx",
			"tag":        "1.25.0",
			"pullPolicy": "IfNotPresent",
		},
		"service": map[string]any{
			"type": "ClusterIP",
			"port": 80,
		},
	}

	// Render-time override values (from engine.WithValues)
	renderTimeValues := map[string]any{
		"replicaCount": 5, // Override replica count
		"image": map[string]any{
			"tag": "1.26.0", // Override only the tag, keep repository and pullPolicy
		},
	}

	// Merge produces final values
	finalValues := util.DeepMerge(configValues, renderTimeValues)

	fmt.Println("replicaCount:", finalValues["replicaCount"].(int)) // 5 (overridden)

	image := finalValues["image"].(map[string]any)
	fmt.Println("image.repository:", image["repository"].(string)) // "nginx" (preserved from config)
	fmt.Println("image.tag:", image["tag"].(string))               // "1.26.0" (overridden)
	fmt.Println("image.pullPolicy:", image["pullPolicy"].(string)) // "IfNotPresent" (preserved from config)

	service := finalValues["service"].(map[string]any)
	fmt.Println("service.type:", service["type"].(string)) // "ClusterIP" (unchanged)
	fmt.Println("service.port:", service["port"].(int))    // 80 (unchanged)

	// Output:
	// replicaCount: 5
	// image.repository: nginx
	// image.tag: 1.26.0
	// image.pullPolicy: IfNotPresent
	// service.type: ClusterIP
	// service.port: 80
}

// Benchmarks

func BenchmarkDeepMerge_SmallMaps(b *testing.B) {
	base := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": map[string]any{
			"nested1": "value",
		},
	}
	overlay := map[string]any{
		"key2": "override",
		"key4": "new",
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_LargeMaps(b *testing.B) {
	base := make(map[string]any, 100)
	for i := range 100 {
		base[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	overlay := make(map[string]any, 50)
	for i := range 50 {
		overlay[fmt.Sprintf("key%d", i*2)] = fmt.Sprintf("override%d", i)
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_DeepNesting(b *testing.B) {
	base := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"level4": map[string]any{
						"level5": "value",
					},
				},
			},
		},
	}
	overlay := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"level4": map[string]any{
						"level5": "override",
						"new":    "value",
					},
				},
			},
		},
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_WithSlices(b *testing.B) {
	base := map[string]any{
		"anySlice":    []any{"a", "b", "c"},
		"stringSlice": []string{"x", "y", "z"},
		"intSlice":    []int{1, 2, 3, 4, 5},
		"config": map[string]any{
			"nested": []any{
				map[string]any{"key": "value"},
			},
		},
	}
	overlay := map[string]any{
		"anySlice":    []any{"d", "e"},
		"stringSlice": []string{"p", "q"},
		"intSlice":    []int{6, 7, 8},
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_ComplexNested(b *testing.B) {
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
			"annotations": map[string]any{
				"key1": "value1",
			},
		},
		"resources": map[string]any{
			"limits": map[string]any{
				"cpu":    "100m",
				"memory": "128Mi",
			},
		},
	}
	overlay := map[string]any{
		"replicaCount": 3,
		"image": map[string]any{
			"tag": "v2.0",
		},
		"service": map[string]any{
			"type": "LoadBalancer",
			"annotations": map[string]any{
				"key2": "value2",
			},
		},
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_TypedSlices(b *testing.B) {
	b.Run("string", func(b *testing.B) {
		base := map[string]any{
			"tags": []string{"dev", "test", "staging"},
		}
		overlay := map[string]any{
			"tags": []string{"prod"},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(base, overlay)
		}
	})

	b.Run("int", func(b *testing.B) {
		base := map[string]any{
			"ports": []int{8080, 9090, 3000},
		}
		overlay := map[string]any{
			"ports": []int{443},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(base, overlay)
		}
	})

	b.Run("int64", func(b *testing.B) {
		base := map[string]any{
			"timestamps": []int64{1234567890, 9876543210},
		}
		overlay := map[string]any{
			"timestamps": []int64{1111111111},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(base, overlay)
		}
	})

	b.Run("float64", func(b *testing.B) {
		base := map[string]any{
			"metrics": []float64{1.1, 2.2, 3.3},
		}
		overlay := map[string]any{
			"metrics": []float64{4.4},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(base, overlay)
		}
	})

	b.Run("bool", func(b *testing.B) {
		base := map[string]any{
			"flags": []bool{true, false, true},
		}
		overlay := map[string]any{
			"flags": []bool{false},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(base, overlay)
		}
	})
}

func BenchmarkDeepMerge_NilHandling(b *testing.B) {
	b.Run("bothNil", func(b *testing.B) {
		for range b.N {
			_ = util.DeepMerge(nil, nil)
		}
	})

	b.Run("baseNil", func(b *testing.B) {
		overlay := map[string]any{
			"key1": "value1",
			"key2": map[string]any{"nested": "value"},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(nil, overlay)
		}
	})

	b.Run("overlayNil", func(b *testing.B) {
		base := map[string]any{
			"key1": "value1",
			"key2": map[string]any{"nested": "value"},
		}
		b.ResetTimer()
		for range b.N {
			_ = util.DeepMerge(base, nil)
		}
	})
}

func BenchmarkDeepMerge_TypeMismatch(b *testing.B) {
	base := map[string]any{
		"field1": map[string]any{
			"nested": "value",
		},
		"field2": "string",
		"field3": []any{"a", "b"},
	}
	overlay := map[string]any{
		"field1": "now_a_string",
		"field2": map[string]any{"now": "map"},
		"field3": 123,
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_NoOverlap(b *testing.B) {
	base := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	overlay := map[string]any{
		"key4": "value4",
		"key5": "value5",
		"key6": "value6",
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}

func BenchmarkDeepMerge_FullOverlap(b *testing.B) {
	base := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	overlay := map[string]any{
		"key1": "override1",
		"key2": "override2",
		"key3": "override3",
	}

	for b.Loop() {
		_ = util.DeepMerge(base, overlay)
	}
}
