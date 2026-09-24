package model

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUserModelLockedFindsUseForUpdate(t *testing.T) {
	var output bytes.Buffer
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "user:password@tcp(localhost:3306)/chasing_points?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger: logger.New(log.New(&output, "", 0), logger.Config{
			LogLevel: logger.Info,
		}),
	})
	if err != nil {
		t.Fatalf("open MySQL dry-run DB: %v", err)
	}

	users := NewUserModel(db)
	if _, err := users.FindByIdForUpdateWithTx(db, 42); err != nil {
		t.Fatalf("find user by id with lock: %v", err)
	}
	if _, err := users.FindByPhoneForUpdateWithTx(db, "13800138000"); err != nil {
		t.Fatalf("find user by phone with lock: %v", err)
	}

	if count := strings.Count(output.String(), "FOR UPDATE"); count != 2 {
		t.Fatalf("expected two locking reads, got %d: %s", count, output.String())
	}
}
