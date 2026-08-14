package middleware

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"chasing_points/internal/observability"

	"github.com/zeromicro/go-zero/core/logx"
)

const requestIDHeader = "X-Request-ID"

var requestSequence atomic.Uint64

type RequestCompletion struct {
	RequestID         string
	Method            string
	Route             string
	Status            int
	Duration          time.Duration
	ResponseBytes     int64
	SQLCount          int64
	SQLDuration       time.Duration
	SlowSQLCount      int64
	CacheHits         uint64
	CacheMisses       uint64
	CacheDecodeErrors uint64
	CacheRedisErrors  uint64
	CacheWriteErrors  uint64
	CacheFallbacks    uint64
}

// RequestObservabilityMiddleware emits one compact structured record for every
// completed HTTP request. It intentionally uses the path only (never the query
// string) and normalizes opaque numeric/UUID segments to keep log labels bounded.
func RequestObservabilityMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return NewRequestObservabilityMiddleware(logRequestCompletion)(next)
}

func NewRequestObservabilityMiddleware(observer func(context.Context, RequestCompletion)) func(http.HandlerFunc) http.HandlerFunc {
	if observer == nil {
		observer = func(context.Context, RequestCompletion) {}
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			requestID := resolveRequestID(r.Header.Get(requestIDHeader))
			w.Header().Set(requestIDHeader, requestID)

			ctx := observability.WithRequestID(r.Context(), requestID)
			ctx = observability.WithRequestMetrics(ctx, observability.NewRequestMetrics(time.Now()))
			ctx = logx.ContextWithFields(ctx, logx.Field("request_id", requestID))
			startedAt := time.Now()
			recorder := &responseMetricsWriter{ResponseWriter: w}
			recordCompletion := func(status int) {
				snapshot := observability.RequestMetricsFromContext(ctx).Snapshot()
				observer(ctx, RequestCompletion{
					RequestID:         requestID,
					Method:            r.Method,
					Route:             normalizeRoute(r.URL.Path),
					Status:            status,
					Duration:          time.Since(startedAt),
					ResponseBytes:     recorder.Bytes(),
					SQLCount:          snapshot.SQLCount,
					SQLDuration:       snapshot.SQLDuration,
					SlowSQLCount:      snapshot.SlowQueryCount,
					CacheHits:         snapshot.CacheHits,
					CacheMisses:       snapshot.CacheMisses,
					CacheDecodeErrors: snapshot.CacheDecodeErrors,
					CacheRedisErrors:  snapshot.CacheRedisErrors,
					CacheWriteErrors:  snapshot.CacheWriteErrors,
					CacheFallbacks:    snapshot.CacheFallbacks,
				})
			}
			defer func() {
				if recovered := recover(); recovered != nil {
					recordCompletion(http.StatusInternalServerError)
					panic(recovered)
				}
				recordCompletion(recorder.Status())
			}()

			next(recorder, r.WithContext(ctx))
		}
	}
}

func logRequestCompletion(ctx context.Context, completion RequestCompletion) {
	logger := logx.WithContext(ctx)
	logger.Infow("http_request_completed",
		logx.Field("method", completion.Method),
		logx.Field("route", completion.Route),
		logx.Field("status", completion.Status),
		logx.Field("duration_ms", completion.Duration.Milliseconds()),
		logx.Field("response_bytes", completion.ResponseBytes),
		logx.Field("sql_count", completion.SQLCount),
		logx.Field("sql_duration_ms", completion.SQLDuration.Milliseconds()),
		logx.Field("slow_sql_count", completion.SlowSQLCount),
		logx.Field("cache_hits", completion.CacheHits),
		logx.Field("cache_misses", completion.CacheMisses),
		logx.Field("cache_decode_errors", completion.CacheDecodeErrors),
		logx.Field("cache_redis_errors", completion.CacheRedisErrors),
		logx.Field("cache_write_errors", completion.CacheWriteErrors),
		logx.Field("cache_fallbacks", completion.CacheFallbacks),
	)
}

func resolveRequestID(value string) string {
	value = strings.TrimSpace(value)
	if isSafeRequestID(value) {
		return value
	}
	return fmt.Sprintf("req-%x-%x", time.Now().UnixNano(), requestSequence.Add(1))
}

func isSafeRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func normalizeRoute(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	segments := strings.Split(path, "/")
	for index, segment := range segments {
		if isOpaqueRouteSegment(segment) {
			segments[index] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

func isOpaqueRouteSegment(segment string) bool {
	if segment == "" {
		return false
	}
	allDigits := true
	for _, char := range segment {
		if !unicode.IsDigit(char) {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}

	if len(segment) != 36 {
		return false
	}
	for index, char := range segment {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

type responseMetricsWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *responseMetricsWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseMetricsWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	count, err := w.ResponseWriter.Write(data)
	w.bytes += int64(count)
	return count, err
}

func (w *responseMetricsWriter) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *responseMetricsWriter) Bytes() int64 {
	return w.bytes
}

func (w *responseMetricsWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseMetricsWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *responseMetricsWriter) Push(target string, options *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}

func (w *responseMetricsWriter) ReadFrom(reader io.Reader) (int64, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		count, err := readerFrom.ReadFrom(reader)
		w.bytes += count
		return count, err
	}
	return io.Copy(writerOnly{w}, reader)
}

func (w *responseMetricsWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type writerOnly struct {
	writer io.Writer
}

func (w writerOnly) Write(data []byte) (int, error) {
	return w.writer.Write(data)
}
