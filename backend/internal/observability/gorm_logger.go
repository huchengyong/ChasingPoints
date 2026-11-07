package observability

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm/logger"
)

const defaultSlowSQLThreshold = 500 * time.Millisecond

var (
	sqlStringLiteralPattern      = regexp.MustCompile(`'(?:''|[^'])*'`)
	sqlDoubleQuotedStringPattern = regexp.MustCompile(`"(?:""|[^"])*"`)
	sqlNumberLiteralPattern      = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
)

// GormLogger records request-scoped query totals and only emits structured SQL
// logs for errors and slow statements. SQL values are removed before logging.
type GormLogger struct {
	slowThreshold time.Duration
}

func NewGormLogger(slowThreshold time.Duration) GormLogger {
	if slowThreshold <= 0 {
		slowThreshold = defaultSlowSQLThreshold
	}
	return GormLogger{slowThreshold: slowThreshold}
}

func (l GormLogger) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (GormLogger) Info(context.Context, string, ...interface{}) {}

func (GormLogger) Warn(context.Context, string, ...interface{}) {}

func (GormLogger) Error(context.Context, string, ...interface{}) {}

func (l GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	slow := elapsed >= l.slowThreshold
	ObserveSQL(ctx, elapsed, slow)

	if (err == nil || errors.Is(err, logger.ErrRecordNotFound)) && !slow {
		return
	}

	sql, rows := fc()
	fields := []logx.LogField{
		logx.Field("duration_ms", elapsed.Milliseconds()),
		logx.Field("rows_affected", rows),
		logx.Field("sql_template", normalizeSQL(sql)),
	}
	if err != nil && !errors.Is(err, logger.ErrRecordNotFound) {
		fields = append(fields, logx.Field("error_category", classifyDatabaseError(err)))
		logx.WithContext(ctx).Errorw("sql_query_failed", fields...)
		return
	}
	logx.WithContext(ctx).Sloww("slow_sql", fields...)
}

func normalizeSQL(sql string) string {
	sql = sqlStringLiteralPattern.ReplaceAllString(sql, "?")
	sql = sqlDoubleQuotedStringPattern.ReplaceAllString(sql, "?")
	sql = sqlNumberLiteralPattern.ReplaceAllString(sql, "?")
	return strings.Join(strings.Fields(sql), " ")
}

func classifyDatabaseError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "duplicate"):
		return "duplicate"
	case strings.Contains(message, "deadlock"):
		return "deadlock"
	case strings.Contains(message, "timeout"):
		return "timeout"
	case strings.Contains(message, "connection"):
		return "connection"
	default:
		return "database_error"
	}
}
