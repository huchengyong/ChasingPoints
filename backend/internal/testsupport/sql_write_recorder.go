package testsupport

import (
	"context"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm/logger"
)

// SQLWriteRecorder is a silent Gorm logger for read-path tests. Reset it after
// test setup, then assert that the exercised GET logic issued no writes.
type SQLWriteRecorder struct {
	mu     sync.Mutex
	writes []string
}

func NewSQLWriteRecorder() *SQLWriteRecorder {
	return &SQLWriteRecorder{}
}

func (r *SQLWriteRecorder) LogMode(logger.LogLevel) logger.Interface {
	return r
}

func (*SQLWriteRecorder) Info(context.Context, string, ...interface{})  {}
func (*SQLWriteRecorder) Warn(context.Context, string, ...interface{})  {}
func (*SQLWriteRecorder) Error(context.Context, string, ...interface{}) {}

func (r *SQLWriteRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	statement, _ := fc()
	verb := ""
	if fields := strings.Fields(strings.ToUpper(statement)); len(fields) > 0 {
		verb = fields[0]
	}
	switch verb {
	case "INSERT", "UPDATE", "DELETE", "REPLACE", "CREATE", "ALTER", "DROP", "TRUNCATE":
		r.mu.Lock()
		r.writes = append(r.writes, statement)
		r.mu.Unlock()
	}
}

func (r *SQLWriteRecorder) Reset() {
	r.mu.Lock()
	r.writes = nil
	r.mu.Unlock()
}

func (r *SQLWriteRecorder) Writes() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.writes...)
}
