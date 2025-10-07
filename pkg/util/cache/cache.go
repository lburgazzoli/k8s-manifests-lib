package cache

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	utilk8s "github.com/lburgazzoli/k8s-manifests-lib/pkg/util/k8s"
)

const (
	defaultTTL = 5 * time.Minute
)

// Interface is a generic cache interface with TTL-based expiration.
type Interface[T any] interface {
	// Get retrieves a cached value for the given key.
	// Returns the cached value and true if found and not expired, or the zero value and false otherwise.
	Get(key string) (T, bool)

	// Set stores a value for the given key.
	// The entry will automatically expire after the configured TTL.
	Set(key string, value T)

	// Sync removes all expired entries from the cache.
	Sync()
}

type entry[T any] struct {
	value      T
	expiration time.Time
}

// defaultCache is the default implementation of Interface[T].
type defaultCache[T any] struct {
	mu      sync.RWMutex
	entries map[string]entry[T]
	ttl     time.Duration
}

// New creates a new cache with the given options.
// If no TTL is specified, defaults to 5 minutes.
func New[T any](opts ...Option) Interface[T] {
	options := Options{
		TTL: defaultTTL,
	}

	for _, opt := range opts {
		opt.ApplyTo(&options)
	}

	if options.TTL <= 0 {
		options.TTL = defaultTTL
	}

	return &defaultCache[T]{
		entries: make(map[string]entry[T]),
		ttl:     options.TTL,
	}
}

func (c *defaultCache[T]) Get(key string) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, exists := c.entries[key]
	if !exists {
		var zero T
		return zero, false
	}

	if time.Now().After(val.expiration) {
		var zero T
		return zero, false
	}

	return val.value, true
}

func (c *defaultCache[T]) Set(key string, val T) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = entry[T]{
		value:      val,
		expiration: time.Now().Add(c.ttl),
	}
}

func (c *defaultCache[T]) Sync() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, val := range c.entries {
		if now.After(val.expiration) {
			delete(c.entries, key)
		}
	}
}

// renderCache wraps a cache and automatically deep clones unstructured slices on get/set.
type renderCache struct {
	cache Interface[[]unstructured.Unstructured]
}

// NewRenderCache creates a new cache for rendering results with automatic deep cloning.
// Entries are deep cloned when stored and when retrieved to prevent cache pollution.
func NewRenderCache(opts ...Option) Interface[[]unstructured.Unstructured] {
	return &renderCache{
		cache: New[[]unstructured.Unstructured](opts...),
	}
}

func (r *renderCache) Get(key string) ([]unstructured.Unstructured, bool) {
	cached, found := r.cache.Get(key)
	if !found {
		return nil, false
	}

	return utilk8s.DeepCloneUnstructuredSlice(cached), true
}

func (r *renderCache) Set(key string, value []unstructured.Unstructured) {
	r.cache.Set(key, utilk8s.DeepCloneUnstructuredSlice(value))
}

func (r *renderCache) Sync() {
	r.cache.Sync()
}
