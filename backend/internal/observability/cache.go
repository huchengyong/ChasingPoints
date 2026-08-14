package observability

import (
	"context"
	"sync/atomic"
)

// CacheMetrics tracks a bounded set of cache outcomes for one cache domain.
// It stores no keys or request data, so it is safe to expose to tests and
// aggregate into external metrics later.
type CacheMetrics struct {
	hits         atomic.Uint64
	misses       atomic.Uint64
	decodeErrors atomic.Uint64
	redisErrors  atomic.Uint64
	writeErrors  atomic.Uint64
	fallbacks    atomic.Uint64
}

type CacheSnapshot struct {
	Hits         uint64
	Misses       uint64
	DecodeErrors uint64
	RedisErrors  uint64
	WriteErrors  uint64
	Fallbacks    uint64
}

func (m *CacheMetrics) Hit(contexts ...context.Context) {
	m.hits.Add(1)
	observeRequestCache(contexts, func(metrics *RequestMetrics) { metrics.cacheHits.Add(1) })
}

func (m *CacheMetrics) Miss(contexts ...context.Context) {
	m.misses.Add(1)
	observeRequestCache(contexts, func(metrics *RequestMetrics) { metrics.cacheMisses.Add(1) })
}

func (m *CacheMetrics) DecodeError(contexts ...context.Context) {
	m.decodeErrors.Add(1)
	observeRequestCache(contexts, func(metrics *RequestMetrics) { metrics.cacheDecodeErrors.Add(1) })
}

func (m *CacheMetrics) RedisError(contexts ...context.Context) {
	m.redisErrors.Add(1)
	observeRequestCache(contexts, func(metrics *RequestMetrics) { metrics.cacheRedisErrors.Add(1) })
}

func (m *CacheMetrics) WriteError(contexts ...context.Context) {
	m.writeErrors.Add(1)
	observeRequestCache(contexts, func(metrics *RequestMetrics) { metrics.cacheWriteErrors.Add(1) })
}

func (m *CacheMetrics) Fallback(contexts ...context.Context) {
	m.fallbacks.Add(1)
	observeRequestCache(contexts, func(metrics *RequestMetrics) { metrics.cacheFallbacks.Add(1) })
}

func observeRequestCache(contexts []context.Context, observe func(*RequestMetrics)) {
	if len(contexts) == 0 || contexts[0] == nil || observe == nil {
		return
	}
	if metrics := RequestMetricsFromContext(contexts[0]); metrics != nil {
		observe(metrics)
	}
}

func (m *CacheMetrics) Snapshot() CacheSnapshot {
	if m == nil {
		return CacheSnapshot{}
	}
	return CacheSnapshot{
		Hits:         m.hits.Load(),
		Misses:       m.misses.Load(),
		DecodeErrors: m.decodeErrors.Load(),
		RedisErrors:  m.redisErrors.Load(),
		WriteErrors:  m.writeErrors.Load(),
		Fallbacks:    m.fallbacks.Load(),
	}
}

func (m *CacheMetrics) Reset() {
	if m == nil {
		return
	}
	m.hits.Store(0)
	m.misses.Store(0)
	m.decodeErrors.Store(0)
	m.redisErrors.Store(0)
	m.writeErrors.Store(0)
	m.fallbacks.Store(0)
}
