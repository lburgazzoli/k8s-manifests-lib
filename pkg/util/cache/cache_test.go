package cache_test

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util/cache"

	. "github.com/onsi/gomega"
)

func TestCache(t *testing.T) {
	g := NewWithT(t)

	t.Run("should cache and retrieve results", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(5 * time.Minute))

		key := "test-key"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Deployment",
				"metadata": map[string]any{
					"name": "test",
				},
			}},
		}

		// Initially empty
		_, found := c.Get(key)
		g.Expect(found).To(BeFalse())

		// Set value
		c.Set(key, result)

		// Should find it now
		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(HaveLen(1))
		g.Expect(cached[0].GetKind()).To(Equal("Deployment"))
	})

	t.Run("should NOT clone cached results", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(5 * time.Minute))

		key := "clone-test"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Service",
				"metadata": map[string]any{
					"name": "test",
				},
			}},
		}

		c.Set(key, result)

		// Get cached result
		cached1, found1 := c.Get(key)
		g.Expect(found1).To(BeTrue())

		// Modify the cached result
		cached1[0].SetName("modified")

		// Get again - should be affected by previous modification since no deep clone
		cached2, found2 := c.Get(key)
		g.Expect(found2).To(BeTrue())
		g.Expect(cached2[0].GetName()).To(Equal("modified"))
	})

	t.Run("should handle different keys separately", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(5 * time.Minute))

		key1 := "key1"
		key2 := "key2"

		result1 := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Deployment",
				"metadata": map[string]any{
					"name": "deployment",
				},
			}},
		}

		result2 := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Service",
				"metadata": map[string]any{
					"name": "service",
				},
			}},
		}

		c.Set(key1, result1)
		c.Set(key2, result2)

		cached1, found1 := c.Get(key1)
		g.Expect(found1).To(BeTrue())
		g.Expect(cached1[0].GetKind()).To(Equal("Deployment"))

		cached2, found2 := c.Get(key2)
		g.Expect(found2).To(BeTrue())
		g.Expect(cached2[0].GetKind()).To(Equal("Service"))
	})

	t.Run("should expire entries after TTL", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(100 * time.Millisecond))

		key := "expiring-key"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Pod",
				"metadata": map[string]any{
					"name": "pod",
				},
			}},
		}

		c.Set(key, result)

		// Should be cached immediately
		_, found := c.Get(key)
		g.Expect(found).To(BeTrue())

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired now
		_, found = c.Get(key)
		g.Expect(found).To(BeFalse())
	})

	t.Run("should handle empty values", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(5 * time.Minute))

		key := "empty-key"
		result := make([]unstructured.Unstructured, 0)

		c.Set(key, result)

		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(BeEmpty())
	})

	t.Run("should handle nil values", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(5 * time.Minute))

		key := "nil-key"
		var result []unstructured.Unstructured

		c.Set(key, result)

		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(BeNil())
	})

	t.Run("should use default TTL if invalid", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(0))
		g.Expect(c).ToNot(BeNil())

		c = cache.New[[]unstructured.Unstructured](cache.WithTTL(-10 * time.Second))
		g.Expect(c).ToNot(BeNil())
	})

	t.Run("should update existing entry", func(t *testing.T) {
		c := cache.New[[]unstructured.Unstructured](cache.WithTTL(5 * time.Minute))

		key := "update-key"

		result1 := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Deployment",
				"metadata": map[string]any{
					"name": "v1",
				},
			}},
		}

		result2 := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Deployment",
				"metadata": map[string]any{
					"name": "v2",
				},
			}},
		}

		c.Set(key, result1)
		c.Set(key, result2)

		// Should have the updated value
		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("v2"))
	})
}

func TestRenderCache(t *testing.T) {
	g := NewWithT(t)

	t.Run("should cache and retrieve results", func(t *testing.T) {
		c := cache.NewRenderCache(cache.WithTTL(5 * time.Minute))

		key := "test-key"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Deployment",
				"metadata": map[string]any{
					"name": "test",
				},
			}},
		}

		// Initially empty
		_, found := c.Get(key)
		g.Expect(found).To(BeFalse())

		// Set value
		c.Set(key, result)

		// Should find it now
		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(HaveLen(1))
		g.Expect(cached[0].GetKind()).To(Equal("Deployment"))
	})

	t.Run("should automatically clone on Get", func(t *testing.T) {
		c := cache.NewRenderCache(cache.WithTTL(5 * time.Minute))

		key := "clone-get-test"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Service",
				"metadata": map[string]any{
					"name": "test",
				},
			}},
		}

		c.Set(key, result)

		// Get cached result
		cached1, found1 := c.Get(key)
		g.Expect(found1).To(BeTrue())

		// Modify the cached result
		cached1[0].SetName("modified")

		// Get again - should NOT be affected by previous modification due to automatic cloning
		cached2, found2 := c.Get(key)
		g.Expect(found2).To(BeTrue())
		g.Expect(cached2[0].GetName()).To(Equal("test"))
	})

	t.Run("should automatically clone on Set", func(t *testing.T) {
		c := cache.NewRenderCache(cache.WithTTL(5 * time.Minute))

		key := "clone-set-test"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Pod",
				"metadata": map[string]any{
					"name": "original",
				},
			}},
		}

		// Set value
		c.Set(key, result)

		// Modify the original
		result[0].SetName("modified")

		// Get from cache - should have original value due to cloning on Set
		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("original"))
	})

	t.Run("should handle empty values", func(t *testing.T) {
		c := cache.NewRenderCache(cache.WithTTL(5 * time.Minute))

		key := "empty-key"
		result := make([]unstructured.Unstructured, 0)

		c.Set(key, result)

		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(BeEmpty())
	})

	t.Run("should handle nil values", func(t *testing.T) {
		c := cache.NewRenderCache(cache.WithTTL(5 * time.Minute))

		key := "nil-key"
		var result []unstructured.Unstructured

		c.Set(key, result)

		cached, found := c.Get(key)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(BeNil())
	})

	t.Run("should expire entries after TTL", func(t *testing.T) {
		c := cache.NewRenderCache(cache.WithTTL(100 * time.Millisecond))

		key := "expiring-key"
		result := []unstructured.Unstructured{
			{Object: map[string]any{
				"kind": "Pod",
				"metadata": map[string]any{
					"name": "pod",
				},
			}},
		}

		c.Set(key, result)

		// Should be cached immediately
		_, found := c.Get(key)
		g.Expect(found).To(BeTrue())

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Sync to trigger expiration
		c.Sync()

		// Should be expired now
		_, found = c.Get(key)
		g.Expect(found).To(BeFalse())
	})

	t.Run("should handle nil cache gracefully", func(t *testing.T) {
		// Create a renderCache with nil underlying cache
		var rc struct {
			cache.Interface[[]unstructured.Unstructured]
		}

		// Get should return false without panicking
		_, found := rc.Get("test-key")
		g.Expect(found).To(BeFalse())

		// Set should not panic
		rc.Set("test-key", []unstructured.Unstructured{})

		// Sync should not panic
		rc.Sync()
	})
}
