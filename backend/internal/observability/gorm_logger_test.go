package observability

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGormLoggerTracksRequestScopedQueries(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	metrics := NewRequestMetrics(time.Now())
	ctx := WithRequestMetrics(context.Background(), metrics)
	if err := db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		t.Fatalf("execute query: %v", err)
	}

	snapshot := metrics.Snapshot()
	if snapshot.SQLCount != 1 {
		t.Fatalf("sql count = %d, want 1", snapshot.SQLCount)
	}
	if snapshot.SQLDuration < 0 {
		t.Fatalf("sql duration must not be negative: %v", snapshot.SQLDuration)
	}
}

func TestNormalizeSQLRemovesLiteralValues(t *testing.T) {
	got := normalizeSQL("SELECT * FROM users WHERE phone = '13800138000' AND id = 42 AND score = 10.5")
	if strings.Contains(got, "13800138000") || strings.Contains(got, "42") || strings.Contains(got, "10.5") {
		t.Fatalf("sql values were not removed: %q", got)
	}
	if want := "SELECT * FROM users WHERE phone = ? AND id = ? AND score = ?"; got != want {
		t.Fatalf("normalized sql = %q, want %q", got, want)
	}
}
