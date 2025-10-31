package cache

import (
	"time"

	"github.com/lburgazzoli/k8s-manifests-lib/pkg/util"
)

// Option is a generic option for Cache.
type Option = util.Option[Options]

// Options is a struct-based option that can set cache options.
type Options struct {
	// TTL is the time-to-live for cache entries.
	TTL time.Duration
}

// ApplyTo applies the cache options to the target configuration.
func (opts Options) ApplyTo(target *Options) {
	if opts.TTL > 0 {
		target.TTL = opts.TTL
	}
}

// WithTTL sets the time-to-live for cache entries.
func WithTTL(ttl time.Duration) Option {
	return util.FunctionalOption[Options](func(opts *Options) {
		opts.TTL = ttl
	})
}
