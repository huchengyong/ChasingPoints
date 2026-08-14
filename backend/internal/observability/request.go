package observability

import (
	"context"
	"sync/atomic"
	"time"
)

type requestMetricsKey struct{}
type requestIDKey struct{}

// RequestMetrics contains request-scoped counters that database callbacks and
// HTTP middleware can update without sharing state across requests.
type RequestMetrics struct {
	startedAt         time.Time
	sqlCount          atomic.Int64
	sqlNanos          atomic.Int64
	slowSQL           atomic.Int64
	cacheHits         atomic.Uint64
	cacheMisses       atomic.Uint64
	cacheDecodeErrors atomic.Uint64
	cacheRedisErrors  atomic.Uint64
	cacheWriteErrors  atomic.Uint64
	cacheFallbacks    atomic.Uint64
}

type RequestSnapshot struct {
	SQLCount          int64
	SQLDuration       time.Duration
	SlowQueryCount    int64
	CacheHits         uint64
	CacheMisses       uint64
	CacheDecodeErrors uint64
	CacheRedisErrors  uint64
	CacheWriteErrors  uint64
	CacheFallbacks    uint64
}

func NewRequestMetrics(startedAt time.Time) *RequestMetrics {
	return &RequestMetrics{startedAt: startedAt}
}

func WithRequestMetrics(ctx context.Context, metrics *RequestMetrics) context.Context {
	return context.WithValue(ctx, requestMetricsKey{}, metrics)
}

func RequestMetricsFromContext(ctx context.Context) *RequestMetrics {
	if ctx == nil {
		return nil
	}
	metrics, _ := ctx.Value(requestMetricsKey{}).(*RequestMetrics)
	return metrics
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

func ObserveSQL(ctx context.Context, duration time.Duration, slow bool) {
	metrics := RequestMetricsFromContext(ctx)
	if metrics == nil {
		return
	}
	metrics.sqlCount.Add(1)
	metrics.sqlNanos.Add(duration.Nanoseconds())
	if slow {
		metrics.slowSQL.Add(1)
	}
}

func (m *RequestMetrics) Snapshot() RequestSnapshot {
	if m == nil {
		return RequestSnapshot{}
	}
	return RequestSnapshot{
		SQLCount:          m.sqlCount.Load(),
		SQLDuration:       time.Duration(m.sqlNanos.Load()),
		SlowQueryCount:    m.slowSQL.Load(),
		CacheHits:         m.cacheHits.Load(),
		CacheMisses:       m.cacheMisses.Load(),
		CacheDecodeErrors: m.cacheDecodeErrors.Load(),
		CacheRedisErrors:  m.cacheRedisErrors.Load(),
		CacheWriteErrors:  m.cacheWriteErrors.Load(),
		CacheFallbacks:    m.cacheFallbacks.Load(),
	}
}
